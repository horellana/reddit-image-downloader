import os
import sys
import json
import asyncio
import argparse
from urllib.parse import urlparse

import aiohttp
from aiofile import AIOFile, Writer


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


def parse_reddit_response(resp):
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


def get_url_filename(url):
    parsed = urlparse(url)

    basename = os.path.basename(parsed.path)
    print(f'Basename: {basename}')

    extension = os.path.splitext(parsed.path)[1]
    print(f'Extension {extension}')

    return [basename, extension]


async def download_image(session, url, folder):
    print(f'Downloading {url} to folder {folder}')

    try:
        async with session.get(url) as response:
            if response.status != 200:
                print(f'Failed to download: {url}')
            else:
                [filename, extension] = get_url_filename(url)

                async with AIOFile(f'{folder}/{filename}', 'wb') as afh:
                    writer = Writer(afh)
                    bytes = await response.read()
                    await writer(bytes)

                    print(f'Correctly downloaded {url}, to folder {folder}')

    except Exception as e:
        print(f'Error: {e}', file=sys.stderr)


async def get_url(session, url):
    async with session.get(url) as response:
        return await response.text()


async def download_subreddit(save_folder, subreddit):
    url = get_subreddit_url(subreddit)

    async with aiohttp.ClientSession() as session:
        response = await get_url(session, url)
        images = parse_reddit_response(json.loads(response))
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

    print(f'args: {args}')

    loop = asyncio.get_event_loop()
    loop.run_until_complete(main(args.folder, args.subreddit))
