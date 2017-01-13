#include <pthread.h>
#include <stdio.h>

int i = 0;


void* inc(){
    for(int j = 0; j < 1000000; j++){
        i++;
    }
    return NULL;
}

void* dec(){
    for (int j = 0; j < 1000000; j++){
        i--;
    }
    return NULL;
}

int main(){
      pthread_t thread_inc;
      pthread_t thread_dec;

      pthread_create(&thread_inc, NULL, inc, NULL);
      pthread_create(&thread_dec, NULL, dec, NULL);

      pthread_join(thread_inc, NULL);
      pthread_join(thread_dec, NULL);

      printf("Result: %d\n", i);

      return 0;
}
