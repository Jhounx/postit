echo "http://testaspnet.vulnweb.com/"  | gauplus | unew -combine  | go run main.go |  xargs -I@ sh -c "dalfox file @ --rawdata --http -X POST" 
