import argparse
import time
import os
import re
import itertools
from multiprocessing import Process, Queue, Value
import logging

import zmq
from util.aima_utils import (
    Expr, expr
)
from util.aima_logic import (
    FolKB, pl_resolution, standardize_variables, parse_definite_clause,
    unify, subst, is_var_symbol, constant_symbols, variables
)
import waggle.plugin as plugin

logging.basicConfig(
    format='%(asctime)s %(levelname)-8s %(message)s',
    level=logging.INFO,
    datefmt='%Y-%m-%d %H:%M:%S')

server_path='/tmp/kb.sock'
event_path='/tmp/event.sock'

goal_status = {}
goal_rules = {}
goal_expressions = {}
measures = {}


def get_rules(goal_id):
    return goal_rules[goal_id]


def get_expressions():
    return goal_expressions


def add_measure(name, value):
    # TODO: we may want to save values with their timestamp
    measures[name.replace(".", "_")] = value


def uniq(array):
    out = []
    for e in array:
        if e not in out:
            out.append(e)
    return out


def gomas_subst(s, x):
    if isinstance(x, list):
        return [gomas_subst(s, xi) for xi in x]
    elif isinstance(x, tuple):
        return tuple([gomas_subst(s, xi) for xi in x])
    elif not isinstance(x, Expr):
        return expr(x)
    elif is_var_symbol(x.op):
        v = s.get(x.op, x)
        if not isinstance(v, Expr):
            return expr(v)
        else:
            return v
    else:
        return Expr(x.op, *[gomas_subst(s, arg) for arg in x.args])


def generate_facts(measures, evaluations):
    permanent_facts = measures
    keys = set(permanent_facts)

    for topic in evaluations:
        if keys.intersection(topic):
            for fact, evaluation, goal_id in evaluations[topic]:
                try:
                    if eval(evaluation, permanent_facts):
                        yield gomas_subst(permanent_facts, expr(fact))
                        # generated_facts.append(subst(permanent_facts, expr(fact)))
                except Exception as ex:
                    # TODO: raise an exeption does not work on the caller
                    #       how to know when it fails to evaluate facts
                    print(str(ex))
                    # raise Exception('Problem in generate_facts: %s' % ('evaluation',))


"""
    Check if FolKB entails the query, alpha
"""
def inferencing(goal_id, alpha):
    # Generate facts based on measures
    clauses = list(generate_facts(measures, goal_expressions))
    clauses.extend(get_rules(goal_id))

    if isinstance(alpha, str):
        alpha = expr(alpha)
    """A simple forward-chaining algorithm. [Figure 9.3]"""
    # TODO: Improve efficiency
    kb_consts = list({c for clause in clauses for c in constant_symbols(clause)})
    def enum_subst(p):
        query_vars = list({v for clause in p for v in variables(clause)})
        for assignment_list in itertools.product(kb_consts, repeat=len(query_vars)):
            theta = {x: y for x, y in zip(query_vars, assignment_list)}
            yield theta

    # check if we can answer without new inferences
    for q in clauses:
        phi = unify(q, alpha, {})
        if phi is not None:
            yield phi

    while True:
        new = []
        for rule in clauses:
            p, q = parse_definite_clause(rule)
            for theta in enum_subst(p):
                if set(subst(theta, p)).issubset(set(clauses)):
                    q_ = subst(theta, q)
                    if all([unify(x, q_, {}) is None for x in clauses + new]):
                        new.append(q_)
                        phi = unify(q_, alpha, {})
                        if phi is not None:
                            yield phi
        if not new:
            break
        for clause in new:
            clauses.append(clause)


def check_triggers(subject, value):
    subject = subject.replace(".", "_")
    for topic in goal_expressions:
        if subject in topic:
            for fact, expression, goal_id in goal_expressions[topic]:
                try:
                    if eval(expression, measures):
                        yield goal_id
                except Exception as ex:
                    print(str(ex))


"""
    check_status_change keeps track of status of goals
    and returns goals that have their status changed
"""
def check_status_change(goal_ids):
    x = expr("x")
    for goal_id in goal_ids:
        if goal_id in goal_status:
            plugin_status = goal_status[goal_id]
        else:
            plugin_status = {}
        plugins_to_run = inferencing(goal_id, "Run(x)")
        for p in plugins_to_run:
            for k, v in p.items():
                if k == x:
                    if v not in plugin_status or plugin_status[v] != "Run":
                        plugin_status[v] = "Run"
                        yield goal_id, "Runnable", str(v)

        plugins_to_stop = inferencing(goal_id, "Stop(x)")
        for p in plugins_to_stop:
            for k, v in p.items():
                if k == x:
                    if v not in plugin_status or plugin_status[v] != "Stop":
                        plugin_status[v] = "Stop"
                        yield goal_id, "Stoppable", str(v)

        goal_status[goal_id] = plugin_status


"""
    extract_variables returns alphabetical words that begin with lowercase
"""
def extract_variables(expr):
    return re.findall(r"\b[a-z]\w+", expr)


"""
    extract_variables_from_predicate_symbol returns variables (like x, y) from
    a predicate symbol (like P, Q, P(x), Q(y))
"""
def extract_variables_from_predicate_symbol(expr):
    return re.findall(r"\b[a-z]", expr)


"""
    handle_rule registers rules that are expressed as first order predicate logic
"""
def handle_rule(msg):
    try:
        goal_id = msg['args'][0]
        rules = msg['args'][1:]
        if goal_id not in goal_rules.keys():
            goal_rules[goal_id] = []
        for rule in rules:
            goal_rules[goal_id].append(expr(rule))
        return True, ''
    except Exception as ex:
        return False, str(ex)


"""
    handle_state extracts variables from an expression and splits
    the expression by ==> that distinguishes variables from the corresponding
    fact. Later, if the expression holds true, then the fact holds true as well.
"""
def handle_state(msg):
    try:
        goal_id = msg['args'][0]
        exprs = msg['args'][1:]
        for expr in exprs:
            # env.detection.smoke to env_detection_smoke
            expr = expr.replace(".", "_")
            variables = extract_variables(expr)
            if variables == []:
                raise Exception("No variable found in {}".format(expr))
            evaluation, fact = expr.strip().split("==>")
            if tuple(variables) not in goal_expressions:
                goal_expressions[tuple(variables)] = []
            goal_expressions[tuple(variables)].append((fact.strip(), evaluation.strip(), goal_id))
        return True, ''
    except Exception as ex:
        return False, str(ex)


def handle_dump(msg):
    try:
        goal_id = msg['args'][0]
        if goal_id in goal_rules:
            del goal_rules[goal_id]
        delete_expr = []
        delete_goals = []
        for e in goal_expressions:
            for expr in goal_expressions[e]:
                _, _, _goal_id = expr
                if goal_id == _goal_id:
                    delete_goals.append(expr)
            for d in delete_goals:
                goal_expressions[e].remove(d)
            if len(goal_expressions[e]) == 0:
                delete_expr.append(e)
        for d in delete_expr:
            del goal_expressions[d]
        return True, ''
    except Exception as ex:
        return False, str(ex)


"""
    handle_ask evaluates a predicate symbol with the knowlgebase and
    returns variables that match the symbol
    # TODO: preciate symbols should have only one variable
            because returned values do not show where they belong (either x or y)
            e.g., Run(x), not Run(x, y)
"""
def handle_ask(msg):
    try:
        goal_id = msg['args'][0]
        expr = msg['args'][1]
        variables = extract_variables_from_predicate_symbol(expr)
        matches = []
        result = inferencing(kb[goal_id], measures, expr)
        for p in result:
            for k, v in p.items():
                for var in variables:
                    if str(k) == var:
                        matches.append(str(v))
                        break
        return write_response(msg, 0, str(matches))
    except Exception as ex:
        return write_response(msg, -1, str(ex))


def handle_measure(msg):
    try:
        name, timestamp, value = msg['args'][:3]
        # TODO: now measures are only numeric values
        if isinstance(value, str):
            value = float(value)
        # TODO: now it keeps the latest
        #       in the future we may need to keep last x measures
        # if name not in measures.keys():
        #     measures[name] = [(timestamp, value)]
        # else:
        add_measure(name, value)
        return True, ''
    except Exception as ex:
        return False, str(ex)


handlers = {
    'rule': handle_rule,
    'state': handle_state,
    'dump': handle_dump,
    'ask': handle_ask,
    'measure': handle_measure,
}


def handle_message(msg):
    logging.info(f"Received {msg['command']}")
    if msg['command'] not in handlers.keys():
        return None
    handler = handlers[msg['command']]
    return handler(msg)


def api_run(message_queue):
    try:
        context = zmq.Context()
        socket = context.socket(zmq.REP)
        socket.bind(f"ipc://{server_path}")
        assert socket

        while True:
            json_msg = socket.recv_json()
            response = {}
            response.update(json_msg)
            if json_msg['command'] == 'ping':
                response['return_code'] = 0
                response['result'] = 'pong'
            else:
                message_queue.put(json_msg)
                response['return_code'] = 0
                response['result'] = 'ack'
            socket.send_json(response)
    except Exception:
        pass
    finally:
        socket.close()


"""
    measure_collector_run consumes sensor readings from the node and
    triggers if an interesting event occurs. The collector ONLY subscribes
    human readable values, not binary blob as a raw sensor measurement simply
    because it does not have a decoder for the raw value.
"""
def rmq_run(message_queue):
    plugin.init()
    # This makes sure it gets ONLY human readable values
    plugin.subscribe("env.#")
    while True:
        measure = plugin.get()
        message_queue.put({
            'command': 'measure',
            'args': [measure.name, measure.timestamp, measure.value]})


def main():
    try:
        message_queue = Queue()
        api_listener = Process(target=api_run, args=(message_queue,))
        rmq_listener = Process(target=rmq_run, args=(message_queue,))

        api_listener.start()
        rmq_listener.start()

        context = zmq.Context()
        socket = context.socket(zmq.PAIR)
        socket.bind(f"ipc://{event_path}")
        assert socket

        while True:
            message = message_queue.get()
            logging.info(f"New message arrived {message}")
            # remote kill command
            if message['command'] == 'terminate':
                break
            r, err = handle_message(message)
            if r is False:
                logging.error(f"Error: {err}")
                continue

            # Examine if the measure triggers any rules
            if message['command'] == 'measure':
                name, timestamp, value = message['args'][:3]
                goals_to_check = list(check_triggers(name, value))
                goals_to_check = uniq(goals_to_check)
                if len(goals_to_check) > 0:
                    events = check_status_change(goals_to_check)
                    for goal_id, status, plugin_name in events:
                        socket.send_json({
                            'goal_id': goal_id,
                            'status': status,
                            'plugin_name': plugin_name
                        })
    except Exception as ex:
        print(str(ex))
    finally:
        api_listener.terminate()
        rmq_listener.terminate()
        socket.close()

if __name__ == '__main__':
    main()
