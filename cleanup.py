import os
import logging
import pathlib
import hashlib

from PIL import Image


logging.basicConfig()


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
            logging.info(f'REMOVE duplicate: {value[0].absolute()}')
            os.remove(value[0].absolute())


def remove_by_resolution(folder):
    files = (path for path in pathlib.Path(folder).iterdir() if path.is_file())

    for file in files:
        try:
            image = Image.open(file.absolute())
            width, height = image.size

            logging.info(f'{file.absolute()} => {width}, {height}')

            if width < 1920 or height < 1080:
                os.remove(file.absolute())
                logging.info('REMOVE wrong resolution')
                continue

            if width / height != 16 / 9:
                os.remove(file.absolute())
                logging.info('REMOVE wrong proportion')
                continue

        except Exception as e:
            logging.error(f'Error while removing files by resolution: {e}')
            logging.error(f'Removing file, because it might be corrupted or invalid!')

            os.remove(file.absolute())
