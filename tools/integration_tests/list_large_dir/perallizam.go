package list_large_dir

import (
	"log"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
)

// wg is used to wait for the program to finish.
var wg sync.WaitGroup

// threadProfile is a profile of threads created by the program.
var threadProfile = pprof.Lookup("threadcreate")

func parallelismInAction(x int, dirPath string, prefix string, t *testing.T) {
	//// lock the current thread.
	//runtime.LockOSThread()
	// defer the call to wg.Done().
	defer wg.Done()
	// sleep
	//time.Sleep(time.Second * 2)

	//for i := ((x - 1) * 1000) + 1; i <= x*1000; i++ {
	//	filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
	//	log.Printf("In %v thread: %v", x, filePath)
	//	_, err := os.Create(filePath)
	//	if err != nil {
	//		t.Errorf("Create file at %q: %v", dirPath, err)
	//	}
	//}

	time.Sleep(time.Second * 120)

	// sleep
	time.Sleep(time.Second * 2)

	// unlock the current thread.
	//runtime.UnlockOSThread()
}

// init is called before main.
// init is predefined function in go like main(), but it is not mandatory to use it.
func init() {
	// set the number of CPUs to use.
	// By default, the number of CPUs is the number of CPUs on the machine.
	// Just to show we can change the number of CPUs
	// I have added this below command.
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func Test(dirPath string, prefix string, t *testing.T) {
	// define the number of goroutines to use.
	count := 12
	// add the number of goroutines to wait for.
	wg.Add(count)
	// log the number of threads created.
	log.Println("Before thread count : ", threadProfile.Count())
	// loop through the number of goroutines.
	for i := 1; i <= count; i++ {
		// call the parallelismInAction function.
		go parallelismInAction(i, dirPath, prefix, t)
	}

	// wait for the goroutines to finish.
	wg.Wait()
	// log the number of threads created.
	log.Println("After thread count : ", threadProfile.Count())
}
