// Copyright 2026 Cloud Droid Hub.
#include <dirent.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/resource.h>

#include "syscall/syscall_wrapper.h"

#ifdef TEST_OLD
#define benchOpenRaw benchmark_syscall_openat
#define benchOpenLib benchmark_libc_openat
#endif

static int countFds() {
    DIR *dir = opendir("/proc/self/fd");
    if (!dir) return -1;
    int count = 0;
    while (dirent *item = readdir(dir)) {
        if (strcmp(item->d_name, ".") && strcmp(item->d_name, "..")) ++count;
    }
    closedir(dir);
    return count;
}

int main(int argc, char **argv) {
    if (argc != 4) return 2;
    const bool raw = !strcmp(argv[1], "raw");
    if (!raw && strcmp(argv[1], "lib")) return 2;
    const int calls = atoi(argv[2]);
    const int cap = atoi(argv[3]);
    if (cap < 16 || cap > 32768) return 2;
    rlimit limit = {static_cast<rlim_t>(cap), static_cast<rlim_t>(cap)};
    if (setrlimit(RLIMIT_NOFILE, &limit)) return 2;
    const int begin = countFds();
    if (begin < 0) return 2;
    int trace[5] = {};
    long long times[5] = {};
    for (int round = 0; round < 5; ++round) {
        times[round] = raw ? benchOpenRaw(calls) : benchOpenLib(calls);
        trace[round] = countFds();
        if (trace[round] < 0) return 2;
    }
    const int end = countFds();
    printf("{\"begin\":%d,\"end\":%d,\"limit\":%d,\"trace\":[", begin, end, cap);
    for (int i = 0; i < 5; ++i) printf("%s%d", i ? "," : "", trace[i]);
    printf("],\"times\":[");
    for (int i = 0; i < 5; ++i) printf("%s%lld", i ? "," : "", times[i]);
    printf("]}\n");
    return end == begin ? 0 : 1;
}
