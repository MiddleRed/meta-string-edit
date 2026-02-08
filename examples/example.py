#!/usr/bin/env python3
"""
Equivalent to examples/example.go

Run from the project root:
    python examples/example.py
"""

from __future__ import annotations

import os
import sys

# Ensure project root is importable
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from metastringedit import MetadataFile

METADATA_FILE_PATH = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    "test",
    "global-metadata.dat.sample1",
)


def main() -> None:
    print("=== Example 1: Load and Inspect ===")
    load_and_inspect()

    print("\n=== Example 2: Search Strings ===")
    search_strings()

    print("\n=== Example 3: Modify Strings ===")
    modify_strings()

    print("\n=== Example 4: Export to JSON ===")
    export_to_json()


# Example 1: Load and inspect a metadata file
def load_and_inspect() -> None:
    mf = MetadataFile(METADATA_FILE_PATH)

    # Display basic information
    print(f"Metadata Version: {mf.get_version()}")
    print(
        f"File Size: {mf.get_file_size()} bytes "
        f"({mf.get_file_size() / 1024 / 1024:.2f} MB)"
    )
    print(f"String Count: {mf.get_string_count()}")

    # First few strings
    print("\nFirst 5 strings:")
    strings = mf.list_strings()
    for s in strings[:5]:
        value = s.value[:57] + "..." if len(s.value) > 60 else s.value
        print(f"  [{s.nth}] {value}")

    mf.close()


# Example 2: Search for strings
def search_strings() -> None:
    mf = MetadataFile(METADATA_FILE_PATH)

    # Normal search (case-insensitive substring)
    print("Searching for 'element'...")
    results = mf.search_strings("element")
    print(f"Found {len(results)} matches (showing first 5):")
    for s in results[:5]:
        value = s.value[:57] + "..." if len(s.value) > 60 else s.value
        print(f"  [{s.nth}] {value}")

    # Regex search
    print("\nSearching with regex '^[A-Z][a-z]+Type$'...")
    results = mf.search_strings("^[A-Z][a-z]+Type$", use_regex=True)
    print(f"Found {len(results)} matches:")
    for s in results:
        print(f"  [{s.nth}] {s.value}")

    mf.close()


# Example 3: Modify strings and save
def modify_strings() -> None:
    mf = MetadataFile(METADATA_FILE_PATH)

    # Get original value
    original = mf.get_string(0)
    print(f"Original string [0]: {original.value!r}")

    # Modify the string
    new_value = "MODIFIED_BY_PYTHON_EXAMPLE"
    mf.modify_string(0, new_value)
    print(f"Modified string [0]: {new_value!r}")

    # Modify multiple strings
    modifications = {
        1: "Second_Modified",
        2: "Third_Modified",
    }
    for idx, val in modifications.items():
        mf.modify_string(idx, val)
        print(f"Modified string [{idx}]: {val!r}")

    # Save to a new file
    output_path = os.path.join(
        os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
        "modified_example.dat",
    )
    mf.save(output_path)
    print(f"\nSaved modified metadata to: {output_path}")

    # Verify the changes
    mf2 = MetadataFile(output_path)
    modified = mf2.get_string(0)
    print(f"Verified string [0] in saved file: {modified.value!r}")

    mf.close()
    mf2.close()

    # Cleanup
    os.remove(output_path)


# Example 4: Export to JSON
def export_to_json() -> None:
    mf = MetadataFile(METADATA_FILE_PATH)

    output_path = os.path.join(
        os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
        "strings_example.json",
    )
    mf.export_json(output_path)
    print(f"Exported {mf.get_string_count()} strings to: {output_path}")
    print("You can now view or process the JSON file with other tools.")

    mf.close()

    # Cleanup
    os.remove(output_path)


if __name__ == "__main__":
    main()
