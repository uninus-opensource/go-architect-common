package configzk

import (
	"encoding/json"
	"log"
	"path/filepath"
	"time"

	"github.com/uninus-opensource/uninus-go-architect-common/flags"

	"github.com/samuel/go-zookeeper/zk"
)

type ConfigFormat map[string]string
type ZKresponder func(nodename string, updatedinfo ConfigFormat)

func ZKConnectAndListen(zkHost []string, servicenode string, resp ZKresponder) (res ConfigFormat, err error) {

	c, _, err := zk.Connect(zkHost, time.Second*10)
	if err != nil {
		return nil, err
	}

	// check if the servicenode exists in the zk nodes
	//
	b, _, err := c.Exists(servicenode)
	if err != nil {
		return nil, err
	}
	if !b {
		log.Printf("node missing %s\n", servicenode)
		return nil, nil
	}

	//retrieve the servicenode children and extract the
	//settings from each child:
	//
	childs, _, err := c.Children(servicenode)

	res = make(ConfigFormat)

	for _, v := range childs {
		data, _, err := c.Get(servicenode + "/" + v)

		kvmap := make(ConfigFormat) //{}

		err = json.Unmarshal(data, &kvmap)
		if err != nil {
			log.Println(err)
		}
		for k, vv := range kvmap {
			res[k] = vv
		}
	}

	//repeatedly get the current data of the servicenode.
	//this will place a wait for notification event.
	//must be called repeatedly because zk event notification
	//occurs only once/event.
	//
	chanSig := make(chan string) //contains the path of the node

	//Notification event waiter goroutine:
	//
	zkWaiter := func(zc *zk.Conn, watchPath string) error {
		for {
			//call the zk method, placing wait request:
			//
			_, _, ch, err := zc.GetW(watchPath)
			if err != nil {
				return err
			}

			//wait for notification:
			//
			<-ch

			//send signal to responders
			//
			chanSig <- watchPath
		}
	}

	//initiate waiter goroutines:
	go zkWaiter(c, servicenode)
	go zkWaiter(c, filepath.Dir(servicenode)+flags.ZK_GLOBALS_CONFIG_PATH)

	//Notification event responder goroutine:
	//
	go func(zc *zk.Conn, rsp ZKresponder) {

		for {
			select {
			//receive affected path through the channel
			//
			case nodePath := <-chanSig:

				//respond to notification by requerying zk,
				//to get the new value at nodePath.
				newdata, _, err := zc.Get(nodePath)

				var newPath []string

				err = json.Unmarshal(newdata, &newPath)
				if err != nil {
					log.Println(err)
				}

				//use this value to lookup the subsequent node matching this path
				for _, v := range newPath {
					newdata, _, err = zc.Get(v)
					if err != nil {
						log.Println(err)
					}
					kvmap := ConfigFormat{}

					err = json.Unmarshal(newdata, &kvmap)
					if err != nil {
						log.Println(err)
					}

					//TODO:
					//call the callback function
					//let the subscriber handle the data
					if rsp != nil {
						rsp(v, kvmap)
					}
				}
			}
		}
	}(c, resp)

	return res, err
}

//verbose version:
func ZKConnectAndListenDbg(zkHost []string, servicenode string, resp ZKresponder) (res ConfigFormat, err error) {

	c, _, err := zk.Connect(zkHost, time.Second*10)
	if err != nil {
		return nil, err
	}

	// check if the servicenode exists in the zk nodes
	//
	log.Printf("Verifying node %s ...\n", servicenode)

	b, _, err := c.Exists(servicenode)
	if err != nil {
		return nil, err
	}
	if !b {
		log.Printf("node missing %s\n", servicenode)
		return nil, nil
	}

	log.Printf("node existence verified for %s\n", servicenode)

	//retrieve the servicenode children and extract the
	//settings from each child:
	//
	childs, _, err := c.Children(servicenode)

	res = make(ConfigFormat)

	for _, v := range childs {
		data, _, err := c.Get(servicenode + "/" + v)

		kvmap := ConfigFormat{}

		err = json.Unmarshal(data, &kvmap)
		if err != nil {
			log.Println(err)
		}
		//res[v] = kvmap
		for k, vv := range kvmap {
			res[k] = vv
		}
	}

	//repeatedly get the current data of the servicenode.
	//this will place a wait for notification event.
	//must be called repeatedly because zk event notification
	//occurs only once/event.
	//
	chanSig := make(chan string) //contains the path of the node

	//Notification event waiter goroutine:
	//
	zkWaiter := func(zc *zk.Conn, watchPath string) error {
		for {
			//call the zk method, placing wait request:
			//
			_, _, ch, err := zc.GetW(watchPath)
			if err != nil {
				return err
			}
			log.Println("")
			log.Printf("Waiting for an event in: %s ...\n", watchPath)

			//wait for notification:
			//
			e := <-ch

			//all below will be executed upon e receive event:
			//
			log.Println("===========================================================")
			log.Printf("event notification at: %s\n", e.Path)
			log.Println("===========================================================")

			//send signal to responders
			//
			chanSig <- watchPath
		}
	}

	//initiate waiter goroutines:
	go zkWaiter(c, servicenode)
	go zkWaiter(c, filepath.Dir(servicenode)+flags.ZK_GLOBALS_CONFIG_PATH)

	//Notification event responder goroutine:
	//
	go func(zc *zk.Conn, rsp ZKresponder) {

		for {
			select {
			//receive affected path through the channel
			//
			case nodePath := <-chanSig:

				//respond to notification by requerying zk,
				//to get the new value at nodePath.
				newdata, _, err := zc.Get(nodePath)

				var newPath []string

				err = json.Unmarshal(newdata, &newPath)
				if err != nil {
					log.Println(err)
				}

				//use this value to lookup the subsequent node matching this path
				for _, v := range newPath {
					newdata, _, err = zc.Get(v)
					if err != nil {
						log.Println(err)
					}
					kvmap := ConfigFormat{}

					err = json.Unmarshal(newdata, &kvmap)
					if err != nil {
						log.Println(err)
					}

					log.Printf("...new data at %s: %v\n", newPath, kvmap)

					//TODO:
					//call the callback function
					//let the subscriber handle the data
					if rsp != nil {
						rsp(v, kvmap)
					}
				}
			}
		}
	}(c, resp)

	return res, err
}
