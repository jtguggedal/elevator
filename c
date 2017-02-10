[1mdiff --git a/Ex2/ex2_c b/Ex2/ex2_c[m
[1mindex 03a0cbe..2af78a1 100755[m
Binary files a/Ex2/ex2_c and b/Ex2/ex2_c differ
[1mdiff --git a/Ex2/ex2_c.c b/Ex2/ex2_c.c[m
[1mindex 7836b71..902398f 100644[m
[1m--- a/Ex2/ex2_c.c[m
[1m+++ b/Ex2/ex2_c.c[m
[36m@@ -1,4 +1,4 @@[m
[31m-// gcc -std=gnu99 -Wall -g -o helloworld_c helloworld_c.c -lpthread[m
[32m+[m[32m// gcc -std=gnu99 -Wall -g -o ex2_c ex_2.c -lpthread[m
 [m
 #include <pthread.h>[m
 #include <semaphore.h>[m
[36m@@ -59,7 +59,7 @@[m [mint main(){[m
 [m
     #if USE_SEMAPHORE[m
         sem_unlink("key");[m
[31m-        key = sem_open("key", O_CREAT, NULL, 1);;[m
[32m+[m[32m        sem_init(&key, 0, 1);;[m
 [m
         pthread_create(&thread_inc, NULL, inc, NULL);[m
         pthread_create(&thread_dec, NULL, dec, NULL);[m
[1mdiff --git a/Ex2/ex2_py.py b/Ex2/ex2_py.py[m
[1mindex 01c54e7..4ba5236 100644[m
[1m--- a/Ex2/ex2_py.py[m
[1m+++ b/Ex2/ex2_py.py[m
[36m@@ -5,14 +5,14 @@[m [mi = 0[m
 def thread_inc(key):[m
     global i[m
     for j in range(1000000):[m
[31m-        key.acquire(1)[m
[32m+[m[32m        key.acquire()[m
         i += 1[m
         key.release()[m
 [m
 def thread_dec(key):[m
     global i[m
     for p in range(1000002):[m
[31m-        key.acquire(1)[m
[32m+[m[32m        key.acquire()[m
         i -= 1[m
         key.release()[m
 [m
