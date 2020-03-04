package main

import (
    "io"
    "net"
    "os"
    "os/user"
    "fmt"
    "strings"
    "encoding/csv"
    "path/filepath"

    jsoniter "github.com/json-iterator/go"

    cm "siteResService/src/common"
)

func main() {
    var name string
    if len(os.Args) < 2 {
        fmt.Println("can not get args, exit")

        return
    }

    name = os.Args[1]
    file, err := os.Open("data/" + name)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    defer file.Close()

    reader := csv.NewReader(file)

    // get title
    record, err := reader.Read()
    if err == io.EOF {
        return
    } else if err != nil {
        fmt.Println("Error:", err)
        return
    }
    titles := record
    for {
        record, err = reader.Read()
        if err == io.EOF {
            break
        } else if err != nil {
            fmt.Println("read line Error:", err.Error())
            continue
        }

        //for i := 0; i < len(record); i++ {
        //    fmt.Println(titles[i], ": ", record[i])
        //    //time.Sleep(2 * time.Second)
        //}

        data, _ := jsoniter.Marshal(record[2])

        fmt.Println(titles[2], ": ", string(data))

        var images []cm.ImageInfo
        if err := jsoniter.Unmarshal([]byte(record[2]), &images); err != nil{
            fmt.Println("error: ", err.Error())
            return
        }

        for _, image := range images {
            fmt.Println("image path: ", image.URL)
            index := strings.Index(image.URL, "/")
            imagePath := image.RootPath + image.URL[index :]
            fmt.Println("image path: ", imagePath)
        }

        // executable file's path
        dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
        if err != nil {
            fmt.Println("ads failed, error: ", err.Error())
        }
        fmt.Println(strings.Replace(dir, "\\", "/", -1))

        // current path
        fmt.Println(os.Getwd())


        // get ip
        netInterfaces, err := net.Interfaces()
        if err != nil {
            fmt.Println("net.Interfaces failed, err:", err.Error())
        }

        for i := 0; i < len(netInterfaces); i++ {
            if (netInterfaces[i].Flags & net.FlagUp) != 0 {
                addrs, _ := netInterfaces[i].Addrs()

                for _, address := range addrs {
                    if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
                        if ipnet.IP.To4() != nil {
                            fmt.Println(ipnet.IP.String())
                        }
                    }
                }
            }
        }

        u, err := user.Current()
        fmt.Printf("Gid %s\n", u.Gid)
        fmt.Printf("Uid %s\n", u.Uid)
        fmt.Printf("Username %s\n", u.Username)
        fmt.Printf("Name %s\n", u.Name)
        fmt.Printf("HomeDir %s\n", u.HomeDir)

        //fmt.Println("\n\n\n\n")
        //time.Sleep(5 * time.Second)
    }
}
