import os
import uuid
import json

from flask import Flask, g, request
from tinydb import TinyDB, Query

app = Flask(__name__)

def get_db(database_path='ecr.json'):
    if 'db' not in g:
        g.db = TinyDB(database_path)
    return g.db


def check_app_params(app):
    if not 'name' in app.keys():
        return False, 'No name found'
    if not 'version' in app.keys():
        return False, 'No version found'
    return True, ''


def register_app(app):
    db = get_db()
    app_table = db.table('applications')
    app_query = Query()
    exists = app_table.search(
        (app_query.name == app['name']) &
        (app_query.version == app['version']))
    if len(exists) > 0:
        return '{}:{} is already registered'.format(app['name'], app['version'])
    app_record = {
        'id': str(uuid.uuid4()),
    }
    app_record.update(app)
    if 'profile' in app_record:
        app_record.pop('profile', None)
    app_table.insert(app_record)

    if 'profile' in app.keys():
        profile_table = db.table('profile')
        profiles = json.loads(app['profile'].replace("'", '"'))
        for profile_name, measure in profiles.items():
            measure = measure[0]
            profile_record = {
                'id': app_record['id'],
                'name': app_record['name'],
                'version': app_record['version'],
                'profile': profile_name
            }
            profile_record.update(measure)
            profile_table.insert(profile_record)

    return app_record


@app.route('/register', methods=['GET', 'POST'])
def register():
    if request.method == 'POST':
        app = request.form
    else:
        app = request.args

    result, message = check_app_params(app)
    if result is not False:
        return register_app(app)
    else:
        return message
    return "Registered"


@app.route('/listapp')
def listapp():
    db = get_db()
    app_table = db.table('applications')
    apps = app_table.all()
    out = ""
    for app in apps:
        out += str(app) + '<br>'
    return out


@app.route('/listprofile')
def listprofile():
    db = get_db()
    app_table = db.table('profile')
    apps = app_table.all()
    out = ""
    for app in apps:
        out += str(app) + '<br>'
    return out


@app.route('/getapp', methods=['GET', 'POST'])
def getapp():
    if request.method == 'POST':
        query = request.form
    else:
        query = request.args

    App = Query()
    db = get_db()
    app_table = db.table('applications')
    if 'id' in query:
        result = app_table.search(App.id == query['id'])
    elif 'name' in query and 'version' in query:
        result = app_table.search(
            (App.name == query['name']) &
            (App.version == query['version']))
    else:
        return "No query value given"
    
    if len(result) > 0:
        return result[0]
    else:
        return "Not found"


@app.route('/getprofile', methods=['GET', 'POST'])
def getprofile():
    if request.method == 'POST':
        query = request.form
    else:
        query = request.args

    App = Query()
    db = get_db()
    app_table = db.table('profile')
    if 'id' in query:
        result = app_table.search(App.id == query['id'])
    elif 'name' in query and 'version' in query:
        result = app_table.search(
            (App.name == query['name']) &
            (App.version == query['version']))
    else:
        return "No query value given"
    
    if len(result) > 0:
        if len(result) > 1:
            nested_result = {}
            for profile in result:
                nested_result[profile['profile']] = profile
            return nested_result
        else:
            return result
    else:
        return "Not found"


if __name__ == '__main__':
    app.run()
