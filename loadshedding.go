package chiutil

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

var utilization uint32

// LoadShedding middleware ...
func LoadShedding(utilizationLimit uint32, refreshRate time.Duration) func(http.Handler) http.Handler {
	go monitor(context.Background(), refreshRate)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			utilization := atomic.LoadUint32(&utilization)
			if utilization >= utilizationLimit {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func monitor(ctx context.Context, refreshRate time.Duration) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	t := time.NewTimer(refreshRate)
	defer t.Stop()

	idle0, total0 := parsestat(f)
	for {
		t.Reset(refreshRate)
		select {
		case <-t.C:
			idle1, total1 := parsestat(f)
			idle, total := idle1-idle0, total1-total0
			atomic.StoreUint32(&utilization, uint32(
				100.0*float32(total-idle)/float32(total),
			))
			idle0, total0 = idle1, total1
		case <-ctx.Done():
			return
		}
	}
}

func parsestat(r io.ReaderAt) (idle, total uint64) {
	contents := make([]byte, 4096)
	if _, err := r.ReadAt(contents, 0); err != nil {
		return
	}

	if pos := bytes.Index(contents, []byte("\n")); pos > 0 {
		contents = contents[:pos]
	}

	fields := bytes.Fields(contents)
	if bytes.Compare(fields[0], []byte("cpu")) != 0 {
		return
	}

	for i := 1; i < len(fields); i++ {
		v, err := strconv.ParseUint(string(fields[i]), 10, 64)
		if err != nil {
			return
		}
		if i == 4 {
			idle = v
		}
		total += v
	}
	return
}
