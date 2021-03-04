import unittest
import time
import json
from multiprocessing import Process

import zmq
from kb import main

class TestKB(unittest.TestCase):
    def setUp(self):
        super().setUp()
        self.kb = Process(target=main)
        self.kb.start()
        api_path = '/tmp/kb.sock'
        event_path = '/tmp/event.sock'
        context = zmq.Context()
        self.socket_request = context.socket(zmq.REQ)
        self.socket_request.connect(f'ipc://{api_path}')
        self.socket_event = context.socket(zmq.PAIR)
        self.socket_event.connect(f'ipc://{event_path}')

    # def test_handle_dump(self):
    #     handle_expr(
    #             {"args": [
    #                 "goal01",
    #                 "env.system.time > 10 ==> Daytime(Now)",
    #                 "env.system.time < 10 ==> Nighttime(Now)"
    #         ]})
    #     handle_expr(
    #             {"args": [
    #                 "goal02",
    #                 "env.system.time > 10 ==> Daytime(Now)",
    #                 "env.system.time < 10 ==> Nighttime(Now)"
    #         ]})
    #     self.assertTrue(len(get_expressions()) > 0)
    #     handle_dump(
    #         {"args": ["goal01"]}
    #     )
    #     self.assertTrue(len(get_expressions()) > 0)
    #     handle_dump(
    #         {"args": ["goal02"]}
    #     )
    #     self.assertTrue(len(get_expressions()) == 0)

    def test_trigger(self):
        command_list = [
            {'command': 'rule',
             'args': [
                'goal01',
                'Daytime(Now) ==> Run(Sampler)',
                'Nighttime(Now) ==> Stop(Sampler)',
                '6 <= sys.time.hour <= 18 ==> Daytime(Now)',
                '6 > sys.time.hour or sys.time.hour > 18 ==> Nighttime(Now)',
            ]},
            {'command': 'measure',
             'args': [
                'sys.time.hour',
                time.time_ns(),
                11,
            ]},
        ]
        for command in command_list:
            self.socket_request.send_json(command)
            self.socket_request.recv_json()

        result = self.socket_event.recv_json()
        self.assertDictEqual(
            result,
            {
                'goal_id': "goal01",
                'status': 'Runnable',
                'plugin_name': "Sampler"
            }
        )

        command_list = [
            {'command': 'measure',
             'args': [
                'sys.time.hour',
                time.time_ns(),
                22,
            ]},
        ]
        for command in command_list:
            self.socket_request.send_json(command)
            self.socket_request.recv_json()

        result = self.socket_event.recv_json()
        self.assertDictEqual(
            result,
            {
                'goal_id': "goal01",
                'status': 'Stoppable',
                'plugin_name': "Sampler"
            }
        )

    def tearDown(self):
        super().tearDown()
        self.socket_request.send_json({'command': 'terminate'})
        self.socket_request.close()
        self.socket_event.close()
        # or kill kb forcefully (this may leave kb's subprocesses alive)
        # self.kb.terminate()

if __name__ == '__main__':
    unittest.main()
