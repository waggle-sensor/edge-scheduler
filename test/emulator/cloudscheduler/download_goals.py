import os
import argparse
from urllib.request import urlopen
import json

def download_goals(url) -> dict:
    with urlopen(url) as f:
        return json.loads(f.read())


def save_goals(path: str, vsn: str, goals: dict):
    with open(os.path.join(path, f'{vsn}.json'), "w") as file:
        file.write(json.dumps(goals, indent=4))


def create_goals(args):
    with open(args.node_list, "r") as file:
        for vsn in file:
            vsn = vsn.strip()
            print(f'Fetching science goals for {vsn}...')
            goals = download_goals(f'http://{args.cloudscheduler}/api/v1/goals/{vsn}')
            save_goals(args.output_path, vsn, goals)
            print("Done")
    return 0


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-node-list",
        dest="node_list",
        action="store",
        type=str,
        help="List of nodes to download a science goal"
    )
    parser.add_argument(
        "-cloudscheduler-uri",
        dest="cloudscheduler",
        action="store",
        default="localhost:9770",
        type=str,
        help="Address:port to cloud scheduler"
    )
    parser.add_argument(
        "-output-path",
        dest="output_path",
        action="store",
        required=True,
        type=str,
        help="Path to store node files"
    )
    return_code = create_goals(parser.parse_args())
    if return_code != 0:
        print(parser.print_help())
    exit(return_code)
