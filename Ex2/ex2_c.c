// gcc -std=gnu99 -Wall -g -o helloworld_c helloworld_c.c -lpthread

#include <pthread.h>
#include <semaphore.h>
#include <stdio.h>

#define USE_MUTEX       0
#define USE_SEMAPHORE   1

sem_t * key;
int i = 0;


void* inc(void * mutex){
    for(int j = 0; j < 1000000; j++){
        #if USE_MUTEX
            pthread_mutex_lock(mutex);
            i++;
            pthread_mutex_unlock(mutex);
        #endif

        #if USE_SEMAPHORE
            sem_wait(key);
            i++;
            sem_post(key);
        #endif
    }
    return NULL;
}

void* dec(void * mutex){
    for (int j = 0; j < 1000002; j++){
        #if USE_MUTEX
            pthread_mutex_lock(mutex);
            i--;
            pthread_mutex_unlock(mutex);
        #endif

        #if USE_SEMAPHORE
            sem_wait(key);
            i--;
            sem_post(key);
        #endif
    }
    return NULL;
}

int main(){
    pthread_t thread_inc;
    pthread_t thread_dec;

    #if USE_MUTEX
        pthread_mutex_t mutex;
        pthread_mutex_init(&mutex, NULL);

        pthread_create(&thread_inc, NULL, inc, (void*) &mutex);
        pthread_create(&thread_dec, NULL, dec, (void*) &mutex);
    #endif

    #if USE_SEMAPHORE
        sem_unlink("key");
        key = sem_open("key", O_CREAT, NULL, 1);;

        pthread_create(&thread_inc, NULL, inc, NULL);
        pthread_create(&thread_dec, NULL, dec, NULL);
    #endif

    pthread_join(thread_inc, NULL);
    pthread_join(thread_dec, NULL);

    printf("Result: %d\n", i);

    return 0;
}
