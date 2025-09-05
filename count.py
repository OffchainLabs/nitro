import re
import sys


def count_bad_lines(filename):
    count = 0
    # Regular expression to find "A of total: B fields" and capture A and B
    pattern = re.compile(r"(\d+) of total: (\d+) fields")

    try:
        with open(filename, 'r') as file:
            for line in file:
                match = pattern.search(line)
                if match and int(match.group(1)) > int(match.group(2)):
                    count += 1
    except FileNotFoundError:
        print(f"Error: The file '{filename}' was not found.")
        return -1

    return count


print(count_bad_lines(sys.argv[1]))
