import os
import json
import random
import asyncio
import argparse
from urllib.parse import urlparse

import aiohttp
from aiologger import Logger
from aiofile import AIOFile, Writer

import cleanup

logger = Logger.with_default_handlers(name=__name__)


def has_allowed_file_extensions(url):
    extensions = ['.jpeg',
                  '.png',
                  '.jpg']

    for extension in extensions:
        if extension in url:
            return True

    return False


def get_subreddit_url(subreddit):
    return f'https://www.reddit.com/r/{subreddit}.json'


async def jitter():
    ms = random.randint(500, 1000) / 1000
    await logger.info(f"jitter {ms}")
    await asyncio.sleep(ms)


async def parse_reddit_response(resp):
    await logger.debug(json.dumps(resp, indent=1))

    try:
        result = []

        for child in resp['data']['children']:
            data = child['data']

            item = {
                'title': data['title'],
                'subreddit': data['subreddit'],
                'mature': data['over_18'],
                'url': data['url']
            }

            result.append(item)

        return result

    except Exception as e:
        await logger.error(f'Error while parsing subreddit response: {e}')
        return list()


def get_url_filename(url):
    parsed = urlparse(url)

    basename = os.path.basename(parsed.path)
    extension = os.path.splitext(parsed.path)[1]

    return [basename, extension]


async def download_image(session, url, folder):
    await jitter()

    await logger.info(f'Downloading {url} to folder {folder}')

    try:
        async with session.get(url, timeout=120) as response:
            if response.status != 200:
                await logger.error(f'Failed to download: {url}')
            else:
                [filename, extension] = get_url_filename(url)

                async with AIOFile(f'{folder}/{filename}', 'wb') as afh:
                    writer = Writer(afh)
                    bytes = await response.read()
                    await writer(bytes)
                    await logger.info(f'Correctly downloaded {url}, to folder {folder}')

    except asyncio.exceptions.TimeoutError:
        await logger.error(f'Error while downloading image: {url}, timed out!')

    except Exception as e:
        await logger.error(f'Error while downloading image: {url} , something went wrong ... {e}')


async def get_url(session, url):
    await jitter()

    async with session.get(url) as response:
        return await response.text()


async def download_subreddit(save_folder, subreddit):
    await jitter()

    url = get_subreddit_url(subreddit)

    async with aiohttp.ClientSession() as session:
        response = await get_url(session, url)
        images = await parse_reddit_response(json.loads(response))
        tasks = [download_image(session, image['url'], save_folder)
                 for image in images
                 if has_allowed_file_extensions(image['url'])]

        await asyncio.gather(*tasks)


async def main(save_folder, subreddit):
    tasks = [download_subreddit(args.folder, subreddit) for subreddit in args.subreddit]
    await asyncio.gather(*tasks)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-f', '--folder', help='where to save images', required=True)
    parser.add_argument('-r', '--subreddit',
                        help='subreddit name', nargs='+',
                        required=True)
    args = parser.parse_args()

    loop = asyncio.get_event_loop()
    loop.run_until_complete(main(args.folder, args.subreddit))

    cleanup.remove_duplicates(args.folder)
    cleanup.remove_by_resolution(args.folder)
