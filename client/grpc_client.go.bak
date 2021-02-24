package client

import (
	"log"

	pb "github.com/liuhengloveyou/passport/face/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func UserAdd(cellphone, email, nickname, password string) (userid string, err error) {

	// Set up a connection to the server.
	conn, err := grpc.Dial(":10001", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewPassportRpcClient(conn)

	r, err := c.UserAdd(context.Background(), &pb.UserInfo{Nickname: nickname, Cellphone: cellphone, Email: email})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)

	return "", nil
}

func main() {
	UserAdd("aaa", "aaa", "aaa", "aaa")
}
