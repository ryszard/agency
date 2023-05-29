import ast
import json
import sys
import logging
from io import StringIO

# Set up logging
logging.basicConfig(filename='repl.log', level=logging.DEBUG)

# Create a dictionary to act as the global namespace
globals_dict = {}

def run_code(code):
    old_stdout = sys.stdout
    old_stderr = sys.stderr
    sys.stdout = sys.stderr = mystdout = StringIO()

    try:
        code_ast = ast.parse(code)
        if isinstance(code_ast.body[-1], ast.Expr):
            print_func = ast.Name(id='print', ctx=ast.Load())
            ast.copy_location(print_func, code_ast.body[-1])
            print_call = ast.Call(func=print_func, args=[code_ast.body[-1].value], keywords=[])
            ast.copy_location(print_call, code_ast.body[-1])
            print_expr = ast.Expr(print_call)
            ast.copy_location(print_expr, code_ast.body[-1])
            code_ast.body[-1] = print_expr
            code = compile(code_ast, '<string>', 'exec')

        # Execute the code within the globals_dict namespace
        exec(code, globals_dict)
        return mystdout.getvalue().strip(), ""
    except Exception as e:
        logging.exception("Error while running code")
        return "", str(e)
    finally:
        sys.stdout = old_stdout
        sys.stderr = old_stderr

while True:
    line = sys.stdin.readline()
    if not line:
        break

    data = json.loads(line)
    logging.info("Received data: %s", data)
    
    out, err = run_code(data["code"])

    result = {"out": out, "err": err}
    print(json.dumps(result) + "\n")
    sys.stdout.flush()
