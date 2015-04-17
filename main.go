package main

import (
        "flag"
        "fmt"
        "io"
        "log"
        "net"
        "os"
        "crypto/rand"
)

// Generate a priv/pub keypair
func generate_keypair() (priv, pub [32]byte, err error) {
        _,err = rand.Read(priv[:])
        if err != nil {
                return
        }

        _,err = rand.Read(pub[:])
        if err != nil {
                return
        }
        return
}

// Dial generates a private/public key pair,
// connects to the server, perform the handshake
// and return a reader/writer.
func Dial(addr string) (io.ReadWriteCloser, error) {
        conn, err := net.Dial("tcp", addr)
        if err != nil {
                return nil, err
        }

        //Generate a keypair for the client
        cpriv, cpub, err := generate_keypair()
        if err != nil {
                return nil, err
        }

        //Receive the public key of the server
        var spub [32]byte
        n,err := conn.Read(spub[:])
        if err != nil {
                return nil, err
        }
        fmt.Printf("key received by client(%d) %v\n",n,spub)

        //Send our public key to the server
        _,err = conn.Write(cpub[:])
        if err != nil {
                return nil, err
        }

        secread := NewSecureReader(conn, &cpriv, &spub)
        secwrite := NewSecureWriter(conn, &cpriv, &spub)

        return SecureSocket{
                Reader: secread,
                Writer: secwrite,
                Closer: conn,
                }, err
}

// Serve starts a secure echo server on the given listener.
func Serve(l net.Listener) error {
        conn, err := l.Accept()
        if err != nil {
                return err
        }

        //Generate private and public keys for the server
        spriv, spub, err := generate_keypair()
        if err != nil {
                return err
        }

        //Public key of the client
        var cpub [32]byte

        _,err = conn.Write(spub[:])
        if err != nil {
                return err
        }
        fmt.Printf("key sent from server %v\n",spub)

        _,err = conn.Read(cpub[:])
        if err != nil {
                return err
        }
        fmt.Printf("key received on server %v\n",cpub)

        secread := NewSecureReader(conn, &spriv, &cpub)
        secwrite := NewSecureWriter(conn, &spriv, &cpub)
        buf := make([]byte,1024)
        var n int
        for {
                n,err = secread.Read(buf)
                if err != nil {
                        return err
                }
                _,err = secwrite.Write(buf[:n])
                if err != nil {
                        return err
                }
        }

        return nil
}

func main() {
        port := flag.Int("l", 0, "Listen mode. Specify port")
        flag.Parse()

        // Server mode
        if *port != 0 {
                l, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
                if err != nil {
                        log.Fatal(err)
                }
                defer l.Close()
                log.Fatal(Serve(l))
        }

        // Client mode
        if len(os.Args) != 3 {
                log.Fatalf("Usage: %s <port> <message>", os.Args[0])
        }
        conn, err := Dial("localhost:" + os.Args[1])
        if err != nil {
                log.Fatal(err)
        }
        if _, err := conn.Write([]byte(os.Args[2])); err != nil {
                log.Fatal(err)
        }
        buf := make([]byte, len(os.Args[2]))
        n, err := conn.Read(buf)
        if err != nil && err != io.EOF {
                log.Fatal(err)
        }
        fmt.Printf("%s\n", buf[:n])
}
