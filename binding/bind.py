from __future__ import annotations

import ctypes
import json
import platform
from dataclasses import dataclass
from pathlib import Path
from typing import Union

# Types

FilePath = Union[str, Path]


class MetadataError(Exception):
    """Exception raised by metadata operations."""


@dataclass
class StringLiteral:
    """A string literal entry from the metadata file.

    Attributes:
        nth:   0-based index of the string.
        value: UTF-8 string value.
    """

    nth: int
    value: str


# Shared-library loader (singleton)


def _lib_filename() -> str:
    """Return the platform-specific shared library filename."""
    ext = {"Windows": ".dll", "Linux": ".so", "Darwin": ".dylib"}.get(
        platform.system(), ".so"
    )
    return f"metastringedit{ext}"


def _cdll_get() -> ctypes.CDLL:
    lib_dir = Path(__file__).parent
    lib_path = lib_dir / _lib_filename()
    print(lib_path)
    if not lib_path.exists():
        raise FileNotFoundError("Library is not found, please reinstall the package.")

    lib_path = str(lib_path)

    if platform.system() == "Windows":
        lib = ctypes.CDLL(lib_path, winmode=0)
    else:
        lib = ctypes.CDLL(lib_path)

    return lib


def _load_lib() -> ctypes.CDLL:
    """Load the Go shared library and declare C function signatures."""
    lib = _cdll_get()

    # -- lifecycle -----------------------------------------------------------
    lib.MetaLoadMetadata.argtypes = [ctypes.c_char_p]
    lib.MetaLoadMetadata.restype = ctypes.c_longlong

    lib.MetaFreeMetadata.argtypes = [ctypes.c_longlong]
    lib.MetaFreeMetadata.restype = None

    # -- error / memory ------------------------------------------------------
    lib.MetaGetLastError.argtypes = []
    lib.MetaGetLastError.restype = ctypes.c_void_p

    lib.MetaFreeString.argtypes = [ctypes.c_void_p]
    lib.MetaFreeString.restype = None

    # -- properties ----------------------------------------------------------
    lib.MetaGetVersion.argtypes = [ctypes.c_longlong]
    lib.MetaGetVersion.restype = ctypes.c_int

    lib.MetaGetStringCount.argtypes = [ctypes.c_longlong]
    lib.MetaGetStringCount.restype = ctypes.c_longlong

    lib.MetaGetFileSize.argtypes = [ctypes.c_longlong]
    lib.MetaGetFileSize.restype = ctypes.c_longlong

    # -- string access -------------------------------------------------------
    lib.MetaListStrings.argtypes = [ctypes.c_longlong]
    lib.MetaListStrings.restype = ctypes.c_void_p

    lib.MetaGetString.argtypes = [ctypes.c_longlong, ctypes.c_longlong]
    lib.MetaGetString.restype = ctypes.c_void_p

    lib.MetaSearchStrings.argtypes = [
        ctypes.c_longlong,
        ctypes.c_char_p,
        ctypes.c_int,
    ]
    lib.MetaSearchStrings.restype = ctypes.c_void_p

    # -- mutations -----------------------------------------------------------
    lib.MetaModifyString.argtypes = [
        ctypes.c_longlong,
        ctypes.c_longlong,
        ctypes.c_char_p,
    ]
    lib.MetaModifyString.restype = ctypes.c_int

    lib.MetaSave.argtypes = [ctypes.c_longlong, ctypes.c_char_p]
    lib.MetaSave.restype = ctypes.c_int

    lib.MetaExportJSON.argtypes = [ctypes.c_longlong, ctypes.c_char_p]
    lib.MetaExportJSON.restype = ctypes.c_int

    return lib


_lib: ctypes.CDLL | None = None


def _get_lib() -> ctypes.CDLL:
    """Return the singleton library handle, loading it on first call."""
    global _lib
    if _lib is None:
        _lib = _load_lib()
    return _lib


# Internal helpers


def _read_and_free(lib: ctypes.CDLL, ptr: int | None) -> str | None:
    """Read a C string from *ptr*, free it, and return a Python str."""
    if not ptr:
        return None
    try:
        value = ctypes.string_at(ptr).decode("utf-8")
    finally:
        lib.MetaFreeString(ptr)
    return value


def _last_error(lib: ctypes.CDLL) -> str:
    """Fetch and free the last error message from the library."""
    ptr = lib.MetaGetLastError()
    return _read_and_free(lib, ptr) or "unknown error"


def _parse_string_literals(json_str: str) -> list[StringLiteral]:
    """Deserialise a JSON array into a list of :class:`StringLiteral`."""
    data = json.loads(json_str)
    return [StringLiteral(nth=item["nth"], value=item["value"]) for item in data]


# Public API


class MetadataFile:
    """Handle to a loaded ``global-metadata.dat`` file.

    Mirrors the Go ``*editor.MetadataFile`` API.  Supports the
    context-manager protocol for deterministic cleanup::

        with MetadataFile("path/to/global-metadata.dat") as mf:
            print(mf.get_version())
    """

    # ---- lifecycle ---------------------------------------------------------

    def __init__(self, file_path: FilePath) -> None:
        """Load a metadata file.  Equivalent to ``editor.LoadMetadata``."""
        file_path = str(file_path)

        self._lib = _get_lib()
        self._handle: int | None = None
        self._handle = self._lib.MetaLoadMetadata(file_path.encode("utf-8"))
        if self._handle == -1:
            err = _last_error(self._lib)
            self._handle = None
            raise MetadataError(f"failed to load metadata: {err}")

    def close(self) -> None:
        """Release the underlying Go object."""
        if self._handle is not None:
            self._lib.MetaFreeMetadata(self._handle)
            self._handle = None

    def __enter__(self) -> MetadataFile:
        return self

    def __exit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: object,
    ) -> None:
        self.close()

    def __del__(self) -> None:
        self.close()

    # ---- internal ----------------------------------------------------------

    def _check(self) -> None:
        if self._handle is None:
            raise MetadataError("MetadataFile is closed")

    # ---- properties --------------------------------------------------------

    def get_version(self) -> int:
        """Return the metadata version.  → ``mf.GetVersion()``"""
        self._check()
        return int(self._lib.MetaGetVersion(self._handle))

    def get_string_count(self) -> int:
        """Return the total number of strings.  → ``mf.GetStringCount()``"""
        self._check()
        return int(self._lib.MetaGetStringCount(self._handle))

    def get_file_size(self) -> int:
        """Return the original file size in bytes.  → ``mf.GetFileSize()``"""
        self._check()
        return int(self._lib.MetaGetFileSize(self._handle))

    # ---- string access -----------------------------------------------------

    def list_strings(self) -> list[StringLiteral]:
        """Return every string literal.  → ``mf.ListStrings()``"""
        self._check()
        ptr = self._lib.MetaListStrings(self._handle)
        raw = _read_and_free(self._lib, ptr)
        if raw is None:
            raise MetadataError(f"failed to list strings: {_last_error(self._lib)}")
        return _parse_string_literals(raw)

    def get_string(self, nth: int) -> StringLiteral:
        """Return one string by index.  → ``mf.GetString(nth)``"""
        self._check()
        ptr = self._lib.MetaGetString(self._handle, nth)
        raw = _read_and_free(self._lib, ptr)
        if raw is None:
            raise MetadataError(f"failed to get string: {_last_error(self._lib)}")
        item = json.loads(raw)
        return StringLiteral(nth=item["nth"], value=item["value"])

    def search_strings(
        self, pattern: str, use_regex: bool = False
    ) -> list[StringLiteral]:
        """Search for strings matching *pattern*.  → ``mf.SearchStrings(pattern, useRegex)``

        When *use_regex* is ``True`` the pattern is compiled as a Go
        regular expression; otherwise a case-insensitive substring match
        is performed.
        """  # noqa: E501
        self._check()
        ptr = self._lib.MetaSearchStrings(
            self._handle,
            pattern.encode("utf-8"),
            1 if use_regex else 0,
        )
        raw = _read_and_free(self._lib, ptr)
        if raw is None:
            raise MetadataError(f"search failed: {_last_error(self._lib)}")
        return _parse_string_literals(raw)

    # ---- mutations ---------------------------------------------------------

    def modify_string(self, nth: int, new_value: str) -> None:
        """Replace the string at *nth*.  → ``mf.ModifyString(nth, newValue)``"""
        self._check()
        rc = self._lib.MetaModifyString(self._handle, nth, new_value.encode("utf-8"))
        if rc != 0:
            raise MetadataError(f"failed to modify string: {_last_error(self._lib)}")

    def save(self, output_path: FilePath) -> None:
        """Write the (modified) metadata to *output_path*.  → ``mf.Save(outputPath)``"""
        output_path = str(output_path)
        self._check()
        rc = self._lib.MetaSave(self._handle, output_path.encode("utf-8"))
        if rc != 0:
            raise MetadataError(f"failed to save: {_last_error(self._lib)}")

    def export_json(self, output_path: FilePath) -> None:
        """Export all strings to a JSON file.  → ``mf.ExportJSON(outputPath)``"""
        output_path = str(output_path)
        self._check()
        rc = self._lib.MetaExportJSON(self._handle, output_path.encode("utf-8"))
        if rc != 0:
            raise MetadataError(f"failed to export JSON: {_last_error(self._lib)}")


def load_metadata(file_path: FilePath) -> MetadataFile:
    """Load a ``global-metadata.dat`` file.

    Convenience wrapper equivalent to ``editor.LoadMetadata()`` in Go.
    """
    return MetadataFile(str(file_path))
