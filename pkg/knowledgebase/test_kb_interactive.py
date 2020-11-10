import sys

import zmq

server_path='/tmp/kb.sock'
event_path='/tmp/event.sock'

c = zmq.Context()
socket_api = c.socket(zmq.REQ)
socket_api.connect(f'ipc://{server_path}')
socket_event = c.socket(zmq.PAIR)
socket_event.connect(f'ipc://{event_path}')


def handle_input(line):
    cmd, args = line.strip().split(" ", 1)
    args = args.split(",")
    msg = {
        'command': cmd,
        'args': args
    }
    socket_api.send_json(msg)
    response = socket_api.recv_json()
    print(response)


if len(sys.argv) > 1:
    print(f'Reading commands from {sys.argv[1]}')
    with open(sys.argv[1], 'r') as file:
        for line in file:
            handle_input(line)
else:
    while True:
        line = input("Type commands: ")
        handle_input(line)

socket_api.close()
socket_event.close()
