package rest

import (
	"context"
	"errors"
	"fmt"
	"github.com/breathbath/healthReadyChecks/errs"
	"github.com/breathbath/healthReadyChecks/health"
	"github.com/breathbath/healthReadyChecks/ready"
	"github.com/breathbath/healthReadyChecks/sleep"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

//CloudStorageWriterMock simulates writing data to a cloud storage
type CloudStorageWriterMock struct {
	attemptsCount int
}

//Write simulates writing function which will after 1 attempts simulating storage exhaustion
func (cswm *CloudStorageWriterMock) Write(payload []byte) error {
	cswm.attemptsCount++
	if cswm.attemptsCount > 1 {
		return errors.New("storage exhausted")
	}

	return nil
}

//FileStoreAPI simulates a http API to save files in a cloud storage
type FileStorageAPI struct {
	storage   *CloudStorageWriterMock
	errStream errs.ErrStream
}

//ServeHTTP implements http.Handler interface
func (fsa FileStorageAPI) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := ioutil.ReadAll(r.Body)
	err := fsa.storage.Write(bodyBytes)
	if err != nil {
		fsa.errStream.Send(err)
	}
}

// This shows how to start health rest server as a sidecar and consume errors from a shared error stream
func ExampleServer_Start() {
	//previous declarations
	////CloudStorageWriterMock simulates writing data to a cloud storage
	//type CloudStorageWriterMock struct {
	//	attemptsCount int
	//}
	//
	////Write simulates writing function which will after 1 attempts simulating storage exhaustion
	//func (cswm *CloudStorageWriterMock) Write(payload []byte) error {
	//	cswm.attemptsCount++
	//	if cswm.attemptsCount > 1 {
	//	return errors.New("storage exhausted")
	//}
	//
	//	return nil
	//}
	//
	////FileStoreAPI simulates a http API to save files in a cloud storage
	//type FileStorageAPI struct {
	//	storage *CloudStorageWriterMock
	//	errStream errs.ErrStream
	//}
	//
	////ServeHTTP implements http.Handler interface
	//func (fsa FileStorageAPI) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	//	bodyBytes, _ := ioutil.ReadAll(r.Body)
	//	err := fsa.storage.Write(bodyBytes)
	//	if err != nil {
	//		fsa.errStream.Send(err)
	//	}
	//}

	//starts health server as sidecar with the shared errors stream with the main server, it could be also started as a part of the file server
	startHealthServerHttp := func(ctx context.Context, errStream errs.ErrStream, targetPort, maxErrorsCount int) {
		//we initialise errors listener which will report bad health if errStream will receive more than 1 error per minute
		healthChecker := health.NewErrsListener(maxErrorsCount, time.Minute, errStream)
		go healthChecker.Start(ctx)

		srv := WithHealth(Server{}, healthChecker)

		go func() {
			if err := srv.Start(ctx, targetPort); err != nil {
				log.Fatal(err)
			}
		}()
	}

	//starts simulated file server
	startFileServerHttp := func(errStream errs.ErrStream) *httptest.Server {
		cloudStorage := &CloudStorageWriterMock{}
		apiHandler := FileStorageAPI{
			errStream: errStream,
			storage:   cloudStorage,
		}

		baseSrv := httptest.NewUnstartedServer(apiHandler)
		baseSrv.Start()

		return baseSrv
	}

	sendFileToSave := func(serverAddr string) error {
		req, err := http.NewRequest(
			http.MethodGet,
			"http://"+serverAddr,
			strings.NewReader(""),
		)
		if err != nil {
			return err
		}

		cl := http.Client{}
		resp, err := cl.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	}

	errStream := errs.NewErrStream(10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	healthTargetPort := 8033
	maxErrorsCountPerMinute := 1

	startHealthServerHttp(ctx, errStream, healthTargetPort, maxErrorsCountPerMinute)

	//we've started server which will try to store posted body data in a simulated cloud storage and will start failing after 1 successful requests which should cause the health failure
	srv := startFileServerHttp(errStream)
	defer srv.Close()
	fileServerAddress := srv.Listener.Addr().String()
	healthServerAddress := fmt.Sprintf("http://127.0.0.1:%d", healthTargetPort)

	err := sendFileToSave(fileServerAddress)
	if err != nil {
		log.Fatal(err)
	}

	resp1, err1 := http.Get(healthServerAddress + "/healthz")
	if err1 != nil {
		log.Fatal(err1)
	}
	fmt.Printf("1st sending attempt, the server should be healthy: I am healthy %v\n", resp1.StatusCode == 200)

	err = sendFileToSave(fileServerAddress)
	if err != nil {
		log.Fatal(err)
	}

	resp2, err2 := http.Get(healthServerAddress + "/healthz")
	if err2 != nil {
		log.Fatal(err2)
	}
	fmt.Printf("2nd sending attempt, the server should be healthy, as we expect more than 1 error within a minute: I am healthy: %v\n", resp2.StatusCode == 200)

	err = sendFileToSave(fileServerAddress)
	if err != nil {
		log.Fatal(err)
	}

	resp3, err3 := http.Get(healthServerAddress + "/healthz")
	if err3 != nil {
		log.Fatal(err3)
	}
	fmt.Printf("3d sending attempt, the server should not be healthy, as we have 2 errors within a minute: I am not healthy %v\n", resp3.StatusCode != 200)

	// Output:
	// 1st sending attempt, the server should be healthy: I am healthy true
	// 2nd sending attempt, the server should be healthy, as we expect more than 1 error within a minute: I am healthy: true
	// 3d sending attempt, the server should not be healthy, as we have 2 errors within a minute: I am not healthy true
}

//DbMock simulates a db backend which will be available only after 1 connection attempt
type DbMock struct {
	attemptsCount int
}

//IsAlive simulates health check for a remote db server, if requested for the 2nd time will return true
func (dm *DbMock) IsAlive() bool {
	dm.attemptsCount++
	return dm.attemptsCount > 1
}

//Insert simulates insertion of new data to db in fact it does nothing as it's just an example
func (dm *DbMock) Insert() error {
	return nil
}

//Cache simulates a backend which is always healthy
type Cache struct {}

//Ping simulates another health check which returns error if service is not available
func (c *Cache) Ping() error {
	return nil
}

//Store data in cache
func (c *Cache) Store() error {
	return nil
}

//Read read data from cache
func (c *Cache) Read() (bool, error) {
	return true, nil
}

//ClientApi simulates the end API server which depends on DbMock and Cache so if they both are not healthy then ClientApi will fail ready check
type ClientApi struct {
	Db *DbMock
	Cache *Cache
}

//ServeHTTP implements http.Handler interface, in fact has no meaning for readiness checks but shows a possible implementation for some db/cache driven http API
func (ca ClientApi) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	isFound, err := ca.Cache.Read()
	if err != nil {
		log.Panic(err)
	}

	if isFound {
		return
	}

	err = ca.Db.Insert()
	if err != nil {
		log.Panic(err)
	}
}

//Shows how to use ready http handler which can be added to any http server as part of internal implementation (vs sidecar mode)
func ExampleServer_NewReadyHandler() {
	//having declared models
	////DbMock simulates a db backend which will be available only after 1 connection attempt
	//type DbMock struct {
	//	attemptsCount int
	//}
	//
	////IsAlive simulates health check for a remote db server, if requested for the 2nd time will return true
	//func (dm *DbMock) IsAlive() bool {
	//	dm.attemptsCount++
	//	return dm.attemptsCount > 1
	//}
	//
	////Insert simulates insertion of new data to db in fact it does nothing as it's just an example
	//func (dm *DbMock) Insert() error {
	//	return nil
	//}
	//
	////Cache simulates a backend which is always healthy
	//type Cache struct {}
	//
	////Ping simulates another health check which returns error if service is not available
	//func (c *Cache) Ping() error {
	//	return nil
	//}
	//
	////Store data in cache
	//func (c *Cache) Store() error {
	//	return nil
	//}
	//
	////Read read data from cache
	//func (c *Cache) Read() (bool, error) {
	//	return true, nil
	//}
	//
	////ClientApi simulates the end API server which depends on DbMock and Cache so if they both are not healthy then ClientApi will fail ready check
	//type ClientApi struct {
	//	Db *DbMock
	//	Cache *Cache
	//}
	//
	//ServeHTTP implements http.Handler interface, in fact has no meaning for readiness checks but shows a possible implementation for some db/cache driven http API
	//func (ca ClientApi) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	//	isFound, err := ca.Cache.Read()
	//	if err != nil {
	//		log.Panic(err)
	//	}
	//
	//	if isFound {
	//		return
	//	}
	//
	//	err = ca.Db.Insert()
	//	if err != nil {
	//		log.Panic(err)
	//	}
	//}

	buildReadyHandler := func (db *DbMock, cache *Cache) http.Handler {
		//we create our ready checks against db and cache so client API cannot be ready if dependant services are still pending
		readyChecks := []ready.Test{
	{
		TestFunc: func() error {
		isHealthy := db.IsAlive()
		if !isHealthy {
		return errors.New("db is not ready yet")
	}

		return nil
	},
		Name: "Db Ready Check",
	},
	{
		TestFunc: func() error {
		return cache.Ping()
	},
		Name: "Cache Ready Check",
	},
	}

		readyChecker := ready.NewTestChecker(readyChecks, 1, time.Millisecond, sleep.RuntimeSleeper{})
		readyHttpHandler := NewReadyHandler(time.Second, readyChecker)

		return readyHttpHandler
	}

	//starts simulated client API server, in this case ready check will be part of the main server
	startClientApiHttp := func (db *DbMock, cache *Cache, readyHandler http.Handler) *httptest.Server {
		handleFunc := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/clients" {
		clientAPI := ClientApi{Db: db, Cache: cache}
		clientAPI.ServeHTTP(rw, r)
		return
	}

		if r.URL.Path == "/readyz" {
		readyHandler.ServeHTTP(rw, r)
		return
	}

		rw.WriteHeader(http.StatusNotFound)
	})
		baseSrv := httptest.NewUnstartedServer(handleFunc)
		baseSrv.Start()

		return baseSrv
	}

	db := &DbMock{}
	cache := &Cache{}

	readyHandler := buildReadyHandler(db, cache)

	srv := startClientApiHttp(db, cache, readyHandler)
	apiAddr := srv.Listener.Addr().String()

	resp1, err1 := http.Get("http://" + apiAddr + "/readyz")
	if err1 != nil {
		log.Fatal(err1)
	}
	fmt.Printf("1st ready check, client api should not be ready as DbMock isn't healthy yet: I am not ready: %v\n", resp1.StatusCode != 200)

	resp2, err2 := http.Get("http://" + apiAddr + "/readyz")
	if err2 != nil {
		log.Fatal(err2)
	}
	fmt.Printf("2nd ready check, client api should be ready as DbMock is healthy and Cache is healthy always: I am ready: %v\n", resp2.StatusCode == 200)

	// Output:
	// 1st ready check, client api should not be ready as DbMock isn't healthy yet: I am not ready: true
	// 2nd ready check, client api should be ready as DbMock is healthy and Cache is healthy always: I am ready: true
}
