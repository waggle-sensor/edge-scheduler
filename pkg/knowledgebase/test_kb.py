import unittest
import time
import json

from kb import (
    handle_ping, handle_rule, handle_expr,
    handle_dump, handle_ask, handle_measure,
    check_triggers, add_measure, inferencing,
    get_expressions, check_status_change
)
from util.aima_utils import (
    Expr, expr
)
from waggle.plugin import Message

class TestKB(unittest.TestCase):
    """
        TestKB verifies the KB and its inferencing related functions work
        NOTE: This does NOT test the interface of the KB (i.e., zmq).
              Instead, this directly calls the handler functions
    """
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
        handle_rule(
            {"args": [
                "goal01",
                "Daytime(Now) ==> Run(Cloud)",
                "Daytime(Now) ==> Run(Smoke)",
                "Nighttime(Now) ==> Stop(Cloud)"
        ]})
        handle_expr(
            {"args": [
                "goal01",
                "env.system.time > 10 ==> Daytime(Now)",
                "env.system.time <= 10 ==> Nighttime(Now)"
        ]})
        measure = Message(
            name='env.system.time',
            value=11,
            timestamp=time.time_ns()
        )
        add_measure(measure)
        goals_to_check = list(check_triggers(measure))
        print(goals_to_check)
        events = check_status_change(goals_to_check)
        for e in events:
            print(e)

        events = check_status_change(goals_to_check)
        print("hello")
        for e in events:
            print(e)

        measure = Message(
            name='env.system.time',
            value=9,
            timestamp=time.time_ns()
        )
        add_measure(measure)
        goals_to_check = list(check_triggers(measure))
        print(goals_to_check)
        events = check_status_change(goals_to_check)
        for e in events:
            print(e)

        goals_to_check = list(check_triggers(measure))
        print(goals_to_check)
        events = check_status_change(goals_to_check)
        for e in events:
            print(e)
        # for goal_id in goals_to_check:

            # plugins_to_run = inferencing(goal_id, "Stop(x)")
            # for p in plugins_to_run:
            #     print(p[expr('x')], type(str(p[expr('x')])))
            # print(list(inferencing(goal_id, "Stop(x)")))

if __name__ == '__main__':
    unittest.main()
