import json
from urllib import request, parse
from urllib.parse import urljoin


def add_schedule_in_science_rules(rules):
    new_rules = []
    for rule in rules:
        sp = rule.split(":")
        new_rule = {
            "rule": f'schedule({sp[0]}):{sp[1]}'
        }
        new_rules.append(new_rule)
    return new_rules

for i in range(2, 100, 1):
    ses_management_url = 'http://localhost:19770'

    url = urljoin(ses_management_url, f'api/v1/records/get?id={i}')
    headers={
        "Accept": "application/json"
    }
    req = request.Request(url, headers=headers)
    with request.urlopen(req) as resp:
        job = json.loads(resp.read().decode())

    # removing status attribute in a plugin
    for plugin in job["plugins"]:
        if "status" in plugin:
            plugin.pop("status")

    # add schedule() to the rules
    # job["science_rules"] = add_schedule_in_science_rules(job["science_rules"])

    science_goal = job.get("science_goal", None)
    if science_goal is not None:
        for sub_goal in science_goal["sub_goals"]:
            for plugin in sub_goal["plugins"]:
                if "status" in plugin:
                    plugin.pop("status")
            sub_goal["science_rules"] = add_schedule_in_science_rules(sub_goal["science_rules"])

    url = urljoin(ses_management_url, f'api/v1/records/set?id={i}')
    headers={
        "Accept": "application/json"
    }
    req = request.Request(url, headers=headers, data=json.dumps(job).encode())
    with request.urlopen(req) as resp:
        print(resp.read().decode())

    # print(json.dumps(job, indent=4))