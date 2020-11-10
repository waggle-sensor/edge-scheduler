import itertools

from .aima_utils import (
    Expr, expr
)
from .aima_logic import (
    FolKB, pl_resolution, standardize_variables, parse_definite_clause,
    unify, subst, is_var_symbol, constant_symbols, variables
)

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
    generated_facts = []

    for topic in evaluations:
        if keys.intersection(topic):
            for fact, evaluation in evaluations[topic]:
                try:
                    if eval(evaluation, permanent_facts):
                        generated_facts.append(gomas_subst(permanent_facts, expr(fact)))
                        # generated_facts.append(subst(permanent_facts, expr(fact)))
                except Exception as ex:
                    # TODO: raise an exeption does not work on the caller
                    #       how to know when it fails to evaluate facts
                    print(str(ex))
                    # raise Exception('Problem in generate_facts: %s' % ('evaluation',))
    return generated_facts


#     """
#         Check if FolKB entails the query, alpha
#     """
def inferencing(rules, exprs, measures, alpha):
    # Generate facts based on measures
    clauses = generate_facts(measures, exprs)
    clauses.extend(rules)

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
        # for clause in new:
        #     clauses.append(clause)
