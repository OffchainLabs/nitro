import sys
import os
import glob
import xml.etree.ElementTree as ET

MAX_LENGTH = 2048

def shorten_content(element: ET.Element):
    content = element.text
    if not content:
        return
    if len(content) > MAX_LENGTH:
        content = content[:MAX_LENGTH] + "\n... [CONTENT TRUNCATED]"
    element.text = content


def process_single_file(filepath: str) -> bool:
    print(f"  Processing: {filepath}")
    try:
        tree = ET.parse(filepath)
        root = tree.getroot()

        for elem in root.iter():
            if elem.tag in ['failure']:
                shorten_content(elem)

        tree.write(filepath, encoding='UTF-8', xml_declaration=True)
        return True

    except ET.ParseError as e:
        print(f"  Error parsing XML file {filepath}: {e}", file=sys.stderr)
        return False
    except Exception as e:
        print(f"  An unexpected error occurred processing {filepath}: {e}", file=sys.stderr)
        return False


def process_junit_files(report_dir):
    search_path = os.path.join(report_dir, 'junit*.xml')
    file_paths = glob.glob(search_path)

    if not file_paths:
        print(f"No JUnit XML files found in {report_dir} matching 'junit*.xml'. Exiting gracefully.")
        sys.exit(0)

    print(f"Found {len(file_paths)} JUnit XML files to process.")

    success_count = 0
    for filepath in file_paths:
        if process_single_file(filepath):
            success_count += 1

    print(f"\nProcessing complete: Successfully modified {success_count} of {len(file_paths)} reports.")


if __name__ == '__main__':
    process_junit_files(sys.argv[1])
