import zmq
import time

c = zmq.Context()
s = c.socket(zmq.REQ)

s.connect("ipc:///tmp/kb.sock")

a = {'command': 'measure',
 'args': [
    'env.system.time.hour',
    time.time_ns(),
    1,
]}

s.send_json(a)
s.recv_json()

s.close()
