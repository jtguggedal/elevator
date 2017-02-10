from threading import Thread, Lock

i = 0

def thread_inc(key):
    global i
    for j in range(1000000):
        key.acquire()
        i += 1
        key.release()

def thread_dec(key):
    global i
    for p in range(1000002):
        key.acquire()
        i -= 1
        key.release()

def main():
    key = Lock()
    inc = Thread(target = thread_inc, args = (key,))
    dec = Thread(target = thread_dec, args = (key,))
    inc.start()
    dec.start()
    inc.join()
    dec.join()
    print(i)

main()
