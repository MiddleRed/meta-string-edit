from __future__ import annotations

import os
import tempfile
from pathlib import Path

import pytest

from .bind import MetadataError, MetadataFile

_PROJECT_ROOT = Path(__file__).parent.parent
_TEST_DATA_DIR = _PROJECT_ROOT / "test"
_SAMPLE1 = _TEST_DATA_DIR / "global-metadata.dat.sample1"
_SAMPLE2 = _TEST_DATA_DIR / "global-metadata.dat.sample2"


class TestLoadMetadata:
    """Tests for loading metadata files."""

    def test_load_sample1(self) -> None:
        """TestLoadMetadata_Sample1"""
        with MetadataFile(_SAMPLE1) as mf:
            assert mf.get_version() != 0
            assert mf.get_string_count() > 0
            print(
                f"Sample1: Version={mf.get_version()}, "
                f"StringCount={mf.get_string_count()}, "
                f"FileSize={mf.get_file_size()}"
            )

    def test_load_sample2(self) -> None:
        """TestLoadMetadata_Sample2"""
        with MetadataFile(_SAMPLE2) as mf:
            assert mf.get_version() != 0
            assert mf.get_string_count() > 0
            print(
                f"Sample2: Version={mf.get_version()}, "
                f"StringCount={mf.get_string_count()}, "
                f"FileSize={mf.get_file_size()}"
            )

    def test_load_invalid_file(self) -> None:
        """TestLoadMetadata_InvalidFile"""
        with pytest.raises(MetadataError):
            MetadataFile("nonexistent.dat")


class TestGetVersion:
    def test_version(self) -> None:
        """TestGetVersion"""
        with MetadataFile(_SAMPLE1) as mf:
            version = mf.get_version()
            assert version > 0, f"Invalid version: {version}"


class TestGetStringCount:
    def test_string_count(self) -> None:
        """TestGetStringCount"""
        with MetadataFile(_SAMPLE1) as mf:
            count = mf.get_string_count()
            assert count > 0, f"Invalid string count: {count}"


class TestListStrings:
    def test_list_strings(self) -> None:
        """TestListStrings"""
        with MetadataFile(_SAMPLE1) as mf:
            strings = mf.list_strings()
            assert len(strings) == mf.get_string_count()

            # Verify indices are correct
            for i, s in enumerate(strings):
                assert s.nth == i, f"String at position {i} has incorrect nth: {s.nth}"

            # Log first few strings
            for s in strings[:5]:
                print(f"  [{s.nth}] {s.value!r}")


class TestGetString:
    def test_valid_index(self) -> None:
        """TestGetString — valid index"""
        with MetadataFile(_SAMPLE1) as mf:
            s = mf.get_string(0)
            assert s.nth == 0

    def test_negative_index(self) -> None:
        """TestGetString — negative index"""
        with MetadataFile(_SAMPLE1) as mf:
            with pytest.raises(MetadataError):
                mf.get_string(-1)

    def test_out_of_bounds_index(self) -> None:
        """TestGetString — out of bounds"""
        with MetadataFile(_SAMPLE1) as mf:
            with pytest.raises(MetadataError):
                mf.get_string(mf.get_string_count())


class TestSearchStrings:
    def test_normal_search(self) -> None:
        """TestSearchStrings_Normal — empty pattern matches all"""
        with MetadataFile(_SAMPLE1) as mf:
            results = mf.search_strings("")
            assert len(results) == mf.get_string_count()

    def test_regex_search(self) -> None:
        """TestSearchStrings_Regex"""
        with MetadataFile(_SAMPLE1) as mf:
            # .* should match everything
            results = mf.search_strings(".*", use_regex=True)
            assert len(results) == mf.get_string_count()

            # Invalid regex
            with pytest.raises(MetadataError):
                mf.search_strings("[invalid", use_regex=True)

    def test_no_match(self) -> None:
        """TestSearchStrings_NoMatch"""
        with MetadataFile(_SAMPLE1) as mf:
            results = mf.search_strings("ThisShouldDefinitelyNotExistInMetadata12345")
            assert len(results) == 0


class TestModifyString:
    def test_same_length(self) -> None:
        """TestModifyString_SameLength"""
        with MetadataFile(_SAMPLE1) as mf:
            original = mf.get_string(0)

            # Find a string that is at least 4 chars
            if len(original.value) < 4:
                for i in range(mf.get_string_count()):
                    s = mf.get_string(i)
                    if len(s.value) >= 4:
                        original = s
                        break

            new_value = "x" * len(original.value)
            mf.modify_string(original.nth, new_value)
            modified = mf.get_string(original.nth)
            assert modified.value == new_value

    def test_shorter(self) -> None:
        """TestModifyString_Shorter"""
        with MetadataFile(_SAMPLE1) as mf:
            # Find a string longer than 5 chars
            target_idx = 0
            for i in range(mf.get_string_count()):
                s = mf.get_string(i)
                if len(s.value) > 5:
                    target_idx = i
                    break

            mf.modify_string(target_idx, "short")
            modified = mf.get_string(target_idx)
            assert modified.value == "short"

    def test_longer(self) -> None:
        """TestModifyString_Longer"""
        with MetadataFile(_SAMPLE1) as mf:
            new_value = "This is a much longer string than before"
            mf.modify_string(0, new_value)
            modified = mf.get_string(0)
            assert modified.value == new_value

    def test_invalid_index(self) -> None:
        """TestModifyString_InvalidIndex"""
        with MetadataFile(_SAMPLE1) as mf:
            with pytest.raises(MetadataError):
                mf.modify_string(-1, "test")
            with pytest.raises(MetadataError):
                mf.modify_string(mf.get_string_count(), "test")

    def test_unicode(self) -> None:
        """TestModifyString_Unicode"""
        with MetadataFile(_SAMPLE1) as mf:
            new_value = "Hello 世界 ❤ ✨"
            mf.modify_string(0, new_value)
            modified = mf.get_string(0)
            assert modified.value == new_value


class TestSave:
    def test_no_modifications(self) -> None:
        """TestSave_NoModifications"""
        with MetadataFile(_SAMPLE1) as mf:
            original_count = mf.get_string_count()

            with tempfile.NamedTemporaryFile(suffix=".dat", delete=False) as f:
                tmp_path = f.name
            try:
                mf.save(tmp_path)
                with MetadataFile(tmp_path) as mf2:
                    assert mf2.get_string_count() == original_count
            finally:
                os.unlink(tmp_path)

    def test_with_modifications(self) -> None:
        """TestSave_WithModifications"""
        with MetadataFile(_SAMPLE1) as mf:
            new_value = "Modified String Value"
            mf.modify_string(0, new_value)

            with tempfile.NamedTemporaryFile(suffix=".dat", delete=False) as f:
                tmp_path = f.name
            try:
                mf.save(tmp_path)
                with MetadataFile(tmp_path) as mf2:
                    modified = mf2.get_string(0)
                    assert modified.value == new_value
            finally:
                os.unlink(tmp_path)

    def test_multiple_modifications(self) -> None:
        """TestSave_MultipleModifications"""
        with MetadataFile(_SAMPLE1) as mf:
            modifications = {
                0: "First modification",
                1: "Second modification",
                2: "Third modification",
            }
            for idx, val in modifications.items():
                if idx < mf.get_string_count():
                    mf.modify_string(idx, val)

            with tempfile.NamedTemporaryFile(suffix=".dat", delete=False) as f:
                tmp_path = f.name
            try:
                mf.save(tmp_path)
                with MetadataFile(tmp_path) as mf2:
                    for idx, expected in modifications.items():
                        if idx < mf2.get_string_count():
                            s = mf2.get_string(idx)
                            assert s.value == expected, (
                                f"String {idx}: got {s.value!r}, expected {expected!r}"
                            )
            finally:
                os.unlink(tmp_path)


class TestExportJSON:
    def test_export(self) -> None:
        """TestExportJSON"""
        with MetadataFile(_SAMPLE1) as mf:
            with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
                tmp_path = f.name
            try:
                mf.export_json(tmp_path)
                stat = os.stat(tmp_path)
                assert stat.st_size > 0, "JSON file is empty"
            finally:
                os.unlink(tmp_path)


class TestRoundTrip:
    def test_round_trip(self) -> None:
        """TestRoundTrip: load → modify → save → reload → verify"""
        with MetadataFile(_SAMPLE1) as mf:
            original_count = mf.get_string_count()

            test_data = [
                (0, "Short"),
                (1, "A much longer string than the original one was"),
                (2, "Unicode: 你好世界 🎉"),
            ]

            for idx, value in test_data:
                if idx < original_count:
                    mf.modify_string(idx, value)

            with tempfile.NamedTemporaryFile(suffix=".dat", delete=False) as f:
                tmp_path = f.name
            try:
                mf.save(tmp_path)
                with MetadataFile(tmp_path) as mf2:
                    # Count unchanged
                    assert mf2.get_string_count() == original_count

                    # Values match
                    for idx, expected in test_data:
                        if idx < original_count:
                            s = mf2.get_string(idx)
                            assert s.value == expected, (
                                f"String {idx}: got {s.value!r}, expected {expected!r}"
                            )
            finally:
                os.unlink(tmp_path)
