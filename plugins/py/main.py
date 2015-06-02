#!/usr/bin/env python2
# Asynchronous RPC server over STDIO

from __future__ import print_function
import sys
import time
import pyjsonrpc
import threading
import Queue
import signal
import json
import base64


# --- Lua help text ---

# Help for the functions that are made available to Lua
luahelp = """
add3(number, number) -> number // Adds two numbers and then the number 3
"""

# --- Lua code ---

# $0 is replaced with the path to the plugin, when sending this code to the server
luacode = """
function add3(a, b)
  return CallPlugin("$0", "Add3", a, b)
end
"""

# --- RPC classes and functions ---


def log(*objs):
    """Warning log function that prints to stderr"""
    print("[plugin log]", *objs, file=sys.stderr)


class EncDec:
  """Decorator for decoding and encoding the arguments and return values"""

  def __init__(self, f):
    self.f = f

  def __call__(self, *args, **kwargs):
    a = json.loads(base64.decodestring(args[0]))
    return base64.encodestring(json.dumps(self.f(self, *a)))


class JsonRpc(pyjsonrpc.JsonRpc):
    """Only uppercase methods are made available under the Lua namespace"""

    # Remember to decorate with @EncDec if needed, to enc/dec to base64 and JSON

    @pyjsonrpc.rpcmethod
    @EncDec
    def Add3(self, a, b):
        """Add two numbers and then 3"""
        return a + b + 3

    @pyjsonrpc.rpcmethod
    def Code(self, pluginPath):
      """Return the Lua code for this plugin, as JSON"""
      return luacode.replace("$0", pluginPath)

    @pyjsonrpc.rpcmethod
    def Help(self, args):
      """Return the Lua help for this plugin, as JSON"""
      return luahelp


# --- Common functions ---

queue = Queue.Queue()

def worker(line, q, rpc_client):
    """Worker thread that handles the RPC server calls fror us when requests come in via stdin"""
    out = rpc_client.call(line)
    q.put(out)
    return

def printer(q):
    """Output handler, printer thread will poll the results queue and output results as they appear."""
    #log("Printer started")
    while True:
        out = q.get()
        if out == "kill":
            #log("Kill signal recieved, stopping threads")
            return
        sys.stdout.write(out + "\n")
        sys.stdout.flush()
    return

printer_thread = threading.Thread(target=printer, args=[queue])

def init():
    """Initialise the printer thread and exit signal handler so that we kill long running threads on exit"""

    printer_thread.start()

    def signal_handler(signal, frame):
        queue.put("kill")
        printer_thread.join()
        #sys.exit(0)

    signal.signal(signal.SIGINT, signal_handler)
    return

def main():
    init()
    rpc = JsonRpc()

    # Build the Lua RPC namespace for methods that starts with an uppercase letter
    for name in dir(rpc):
      if name[0].isupper():
        setattr(rpc, "Lua." + name, getattr(rpc, name, None))
       
    line = sys.stdin.readline()

    # The handling of lines is asynchronous,
    # so that out-of-order requests can be handled
    while line:
        try: 
            this_input = line
            t = threading.Thread(target=worker, args=[line, queue, rpc])
            t.start()
            line = sys.stdin.readline()
        except Exception as e:
            log("Exception occured: ", e)
            queue.put("kill")
            printer_thread.join()


if __name__ == "__main__":
    main()
