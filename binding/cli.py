#!/usr/bin/env python3
import ctypes
import sys

from .bind import _cdll_get


def cli_entry():
    lib = _cdll_get()

    py_args = sys.argv
    argc = len(py_args)

    c_argv = (ctypes.c_char_p * argc)(*[arg.encode("utf-8") for arg in py_args])

    lib.RunAsCLI(argc, c_argv)


if __name__ == "__main__":
    cli_entry()
