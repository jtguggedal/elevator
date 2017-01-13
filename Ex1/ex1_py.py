from threading import Thread

i = 0

def increment():
    global i
    for j in range(1000000):
        i += 1

def decrement():
    global i
    for p in range(1000000):
        i -= 1

def main():
	inc = Thread(target = increment, args = (),)
	dec = Thread(target = decrement, args = (),)

	inc.start()
	dec.start()

	inc.join()
	dec.join()
	
	print "Result:", i

main()
