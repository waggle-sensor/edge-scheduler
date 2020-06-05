import sys
import yaml
from urllib import request, parse
from urllib.parse import urljoin

ecr_url = 'http://localhost:5000'
app_definition_file = sys.argv[1] if len(sys.argv) > 1 else 'fake_apps.yaml'

with open(app_definition_file, 'r') as file:
    loaded_yaml = yaml.load(file)
    if 'apps' in loaded_yaml.keys():
        apps = loaded_yaml['apps']
        for app in apps:
            post_data = parse.urlencode(app).encode()
            req = request.Request(urljoin(ecr_url, 'register'), data=post_data)
            res = request.urlopen(req)
            print('Registering {}:{} {}'.format(
                app['name'],
                app['version'],
                res.read().decode()
            ))
