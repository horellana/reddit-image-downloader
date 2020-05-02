import os
import sys
import pathlib
import hashlib
import argparse

from PIL import Image


def hash_file(path):
    with open(path, 'rb') as fh:
        return hashlib.new('md5', fh.read()).hexdigest()


def remove_duplicates(folder):
    things = dict()
    files = (path for path in pathlib.Path(folder).iterdir() if path.is_file())

    for file in files:
        hash = hash_file(file)

        if hash not in things.keys():
            things[hash] = []

        things[hash].append(file)

    for key, value in things.items():
        if len(value) > 1:
            print(f'REMOVE duplicate: {value[0].absolute()}')
            os.remove(value[0].absolute())


def remove_by_resolution(folder):
    files = (path for path in pathlib.Path(folder).iterdir() if path.is_file())
    print(files)

    for file in files:
        try:
            image = Image.open(file.absolute())
            width, height = image.size

            print(f'{file.absolute()} => {width}, {height}')

            if width < 1920 or height < 1080:
                os.remove(file.absolute())
                print('REMOVE wrong resolution')
                continue

            if width / height != 16 / 9:
                os.remove(file.absolute())
                print('REMOVE wrong proportion')
                continue

        except Exception as e:
            print(f'Error while removing files by resolution: {e}', file=sys.stderr)
            print(f'Removing file, because it might be corrupted or invalid!')

            os.remove(file.absolute())


def main(folder):
    remove_duplicates(folder)
    remove_by_resolution(folder)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-f', '--folder',
                        help='where to save images', required=True)

    args = parser.parse_args()

    print(f'args: {args}')

    sys.exit(main(args.folder))
