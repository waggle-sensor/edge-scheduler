import argparse
import yaml
import uuid
import json

from urllib import request, parse
from urllib.parse import urljoin

input_template = """
{
    "user_id": "",
    "goal_name": "",
    "priority": 0,
    "target_nodes": [],
    "rules": [],
    "sensors": [],
    "apps": []
}
"""


def get_template():
    return {
        'header': {
            'goal_id': str(uuid.uuid4()),
            'user_id': '',
            'goal_name': '',
            'priority': 0,
            'target_nodes': []
        },
        'body': {
            'rules': [],
            'sensor_config': '',
            'app_config': []
        }
    }


def read_json(file_path):
    with open(file_path) as json_file:
        request = json.load(json_file)
        assert 'user_id' in request
        assert 'goal_name' in request
        assert 'rules' in request
        assert 'sensors' in request
        assert 'apps' in request
    return request


def call_ecr(url, data, ecr_url='http://localhost:5000'):
    post_data = parse.urlencode(data).encode()
    req = request.Request(urljoin(ecr_url, url), data=post_data)
    response = request.urlopen(req).read()
    return json.loads(response.decode())


def get_app_configuration(app_json):
    assert 'name' in app_json
    assert 'image' in app_json
    app = {'name': app_json['name']}
    app['image'] = app_json['image']

    # query app information from ECR
    _image = app['image'].split('/')[1]
    image, version = _image.split(':')
    query = {'name': image, 'version': version}
    ecr_app = call_ecr('getapp', query)

    # query app profile from ECR
    query = {'id': ecr_app['id']}
    ecr_profile = call_ecr('getprofile', query)
    assert len(ecr_profile.keys()) > 0 and \
           'default' in ecr_profile
    # pulling the resource profile for a specific app mode
    # is not currently supported
    if 'arguments' in app_json:
        app['args'] = str(app_json['arguments'])

    default_resource_limit = ecr_profile['default']
    limits = []
    if 'cpu' in default_resource_limit:
        limits.append({'cpu': default_resource_limit['cpu']})
    if 'memory' in default_resource_limit:
        limits.append({'memory': default_resource_limit['memory']})
    app['resources'] = {'limits': limits}

    # setting other parameters
    if 'ports' in app_json:
        ports = []
        assert type(app_json['ports']) is list
        for port in app_json['ports']:
            ports.append({'containerPort': port})
        app['ports'] = ports
    return app


def do_interactive(): 
    pass


def do_from_file(file_path):
    template = get_template()
    request = read_json(file_path)
    # generate header
    template['header']['user_id'] = request['user_id']
    template['header']['goal_name'] = request['goal_name']
    if 'priority' in request:
        template['header']['priority'] = request['priority']
    if 'target_nodes' in request:
        template['header']['target_nodes'] = request['target_nodes']
    # for sensor in request['sensors']:

    # generate body
    # template['body']['rules'] = request['rules']
    # template['body']['sensors'] = request['sensors']
    app_config = []
    for conditions, apps in request['apps']:
        container = []
        assert type(apps) is list
        for app in apps:
            container.append(get_app_configuration(app))
        containers = {'containers': container}
        app_config.append({str(conditions): {'spec': containers}})
    template['body']['app_config'] = app_config

    print(yaml.dump(template, default_flow_style=False))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-it', '--interactive',
        dest='interactive',
        action='store_true',
        help='enable interactive session')
    parser.add_argument('-f',
        dest='file',
        action='store',
        help='json style user job')
    args = parser.parse_args()
    if args.interactive:
        do_interactive()
    elif args.file:
        do_from_file(args.file)
    else:
        parser.print_help()
