from threading import Thread

i = 0

def thread_inc():
    global i
    for j in range(1000000):
        i += 1

def thread_dec():
    global i
    for p in range(1000000):
        i -= 1

def main():
	inc = Thread(target = thread_inc, args = (),)
	dec = Thread(target = thread_dec, args = (),)
	inc.start()
	dec.start()
	inc.join()
	dec.join()
	print(i)

main()
