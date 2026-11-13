package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)


var (
	db = make(map[string]string)
	dbMu sync.RWMutex
	WriteMu sync.Mutex
)

func main(){
	wal,err:= OpenWAL("wal.log")

	if err != nil{
		log.Fatalf("failed to open WAL: %v", err)
	}

	defer wal.Close()


	// 1. RECOVER PAST STATE:
	log.Println("Scanning WAL and restoring RAM state...")

	records,err:= wal.Recover()

	if err != nil{
		log.Fatalf("recovery failed: %v", err)
	}

	for _,rec:= range records{
		parts:= strings.SplitN(string(rec)," ",3)
		if len(parts)>= 2{
			switch parts[0]{
			case "SET":
				if len(parts) == 3{
					db[parts[1]] = db[parts[2]]
				}

			case "DEL":
				if len(parts) ==2{
					delete(db,parts[1])
				}
			}

		}
	}
	log.Printf("Recovery complete! Restored %d keys into RAM.\n", len(db))


	http.HandleFunc("/set",func(w http.ResponseWriter , r *http.Request){
		key:= r.URL.Query().Get("key")
		val:= r.URL.Query().Get("val")

		if key == "" || val == "" {
			http.Error(w, "missing key or val", http.StatusBadRequest)
			return
		}


		cmd:= fmt.Sprintf("SET %s %s",key,val)

		
		WriteMu.Lock()
		defer WriteMu.Unlock()
		
		err:= wal.Write([]byte(cmd))

		if err != nil{
			http.Error(w, "failed to write to WAL: "+err.Error(), http.StatusInternalServerError)
			return
		}

		//Immediately setting value in the ram
		dbMu.Lock()
		db[key] = val
		dbMu.Unlock()

		fmt.Fprintf(w, "OK: %s = %s\n", key, val)


	})

	http.HandleFunc("/delete",func(w http.ResponseWriter, r *http.Request) {
		key:= r.URL.Query().Get("key")

		if key == " " || key == "" {
			http.Error(w,"missing key",http.StatusBadRequest)
			return
		}		
		
		dbMu.RLock()
		_,exists:= db[key]
		dbMu.RUnlock()
		if !exists{
			http.Error(w,"Value for this key doesn't exist , try another key",http.StatusNotFound)
			return
		}
		
		cmd:= fmt.Sprintf("DEL %s",key)
		
		err:= wal.Write([]byte(cmd))

		if err != nil{
			http.Error(w, "failed to write to WAL: "+err.Error(), http.StatusInternalServerError)
			return
		}

		dbMu.Lock()
		delete(db,key)
		dbMu.Unlock()


		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w)
	})

	http.HandleFunc("/get",func(w http.ResponseWriter, r *http.Request) {
		key:= r.URL.Query().Get("key")


		if key == " " || key == "" {
			http.Error(w,"missing key",http.StatusBadRequest)
			return
		}

		dbMu.RLock()
		defer dbMu.RUnlock()

		val,exists:= db[key]

		if !exists{
			http.Error(w,"Value for this key doesn't exist , try another key",http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w,val)


	})
	log.Println("Server running on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}

}

