import os
import uuid

from flask import Flask, g, request
from tinydb import TinyDB, Query

from pprint import pprint

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
    record = {
        'id': str(uuid.uuid4()),
    }
    record.update(app)
    db.insert(record)
    return True


@app.route('/register', methods=['GET', 'POST'])
def register():
    if request.method == 'POST':
        app = request.form
    else:
        app = request.args

    result, message = check_app_params(app)
    if result is not False:
        register_app(app)
    else:
        return message
    return "Done"


@app.route('/list')
def list():
    db = get_db()
    apps = db.all()
    out = ""
    for app in apps:
        out += str(app) + '<br>'
    return out


@app.route('/get', methods=['GET', 'POST'])
def get():
    db = get_db()

if __name__ == '__main__':
    app.run()
