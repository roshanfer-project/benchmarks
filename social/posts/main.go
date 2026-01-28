package main

import (
	"context"
	"fmt"
	"net"
	"social"
	"strconv"
	"time"

	"social/dagor"
	dagorinit "social/dagor_init"
	pb "social/protobuf"
	rajomoninit "social/rajomon_init"
	"social/utils"

	"github.com/lithammer/shortuuid"
	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
)

const serviceName = "posts"

var log = utils.GetLogger(serviceName)

type PostStorageServer struct {
	pb.UnimplementedPostStorageServer
}

func (s *PostStorageServer) Run() error {
	opts := social.GetServerOptions()
	counter := utils.NewCounterState(serviceName)
	if utils.GetEnvVar("sidecar", false) == "true" && utils.GetEnvVar("queuing_export", false) == "true" {
		opts = append(opts, grpc.UnaryInterceptor(counter.GetInterceptor()))
	} else if utils.GetEnvVar("plain", false) == "true" {
		opts = append(opts, grpc.UnaryInterceptor(counter.GetInterceptor()))
	}

	var priceTable *rajomon.PriceTable = nil
	if utils.GetEnvVar("rajomon", false) == "true" {
		log.Info("rajomon is enabled, configuring rajomon interceptor")
		priceTable = rajomoninit.GetPriceTable(serviceName, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			counter.GetInterceptor(),
			priceTable.UnaryInterceptor,
		))
	}

	var dagorNode *dagor.Dagor = nil
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, false, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			counter.GetInterceptor(),
			dagorNode.UnaryInterceptorServer))
	}

	srv := grpc.NewServer(opts...)

	pb.RegisterPostStorageServer(srv, s)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("PostsPort", true))))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	return srv.Serve(lis)
}

func main() {
	srv := &PostStorageServer{}

	populatePosts(srv,
		utils.StrToInt(utils.GetEnvVar("num_of_posts", true)),
		utils.StrToInt(utils.GetEnvVar("num_of_users", true)))

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func populatePosts(s *PostStorageServer, numberOfPosts, numberOfUser int) {
	for ui := 0; ui < numberOfUser; ui++ {
		userId := "user" + strconv.Itoa(ui)

		posts := make(map[string]interface{}, numberOfPosts)
		postIds := make([]string, numberOfPosts)
		for i := 0; i < numberOfPosts; i++ {
			postId := strconv.Itoa(i)
			timestamp := time.Now().Unix()
			posts[postId] = pb.Post{
				PostId:    postId,
				CreatorId: userId,
				Text:      "This is a sample post",
				Timestamp: timestamp,
			}
			postIds[i] = postId
		}
		utils.SetBulkState(context.Background(), posts)
	}

}

func (s *PostStorageServer) StorePost(ctx context.Context, req *pb.StorePostRequest) (*pb.StorePostResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	utils.BusyLoop(50)
	postId := s.storePost(ctx, req.CreatorId, req.Text)
	return &pb.StorePostResponse{PostId: postId}, nil
}

func (s *PostStorageServer) StorePostMulti(ctx context.Context, req *pb.StorePostMultiRequest) (*pb.StorePostMultiResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	postIds := s.storePostMulti(ctx, req.CreatorId, req.Text, int(req.Number))
	return &pb.StorePostMultiResponse{PostIds: postIds}, nil
}

func (s *PostStorageServer) ReadPost(ctx context.Context, req *pb.ReadPostRequest) (*pb.ReadPostResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	post, err := utils.GetState[pb.Post](ctx, req.PostId)
	if err != nil {
		log.Error("[ReadPost] Error reading post", "postId", req.PostId, "error", err)
		return nil, err
	}
	return &pb.ReadPostResponse{Post: &post}, nil
}

func (s *PostStorageServer) ReadPosts(ctx context.Context, req *pb.ReadPostsRequest) (*pb.ReadPostsResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	retPosts, err := utils.GetBulkState[pb.Post](ctx, req.PostIds)
	if err != nil {
		log.Error("[ReadPosts] Error reading posts", "postIds", req.PostIds, "error", err)
		return nil, err
	}
	posts := make([]*pb.Post, len(retPosts))
	for i, post := range retPosts {
		posts[i] = &post
	}
	return &pb.ReadPostsResponse{Posts: posts}, nil
}

func (s *PostStorageServer) storePost(ctx context.Context, creatorId string, text string) string {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	postIds := s.storePostMulti(ctx, creatorId, text, 1)
	return postIds[0]
}

func (s *PostStorageServer) storePostMulti(ctx context.Context, creatorId string, text string, number int) []string {
	//ctx = config.PropagateMetadata(ctx, "poststorage")
	posts := make(map[string]interface{}, number)
	postIds := make([]string, number)
	for i := 0; i < number; i++ {
		postId := shortuuid.New()
		timestamp := time.Now().Unix()
		posts[postId] = pb.Post{
			PostId:    postId,
			CreatorId: creatorId,
			Text:      text,
			Timestamp: timestamp,
		}
		postIds[i] = postId
	}
	utils.SetBulkState(ctx, posts)
	return postIds
}
