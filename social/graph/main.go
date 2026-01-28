package main

import (
	"context"
	"fmt"
	"net"
	"social"

	"social/dagor"
	dagorinit "social/dagor_init"
	pb "social/protobuf"
	rajomoninit "social/rajomon_init"
	"social/utils"

	"github.com/pennsail/rajomon"
	"google.golang.org/grpc"
)

type GraphServer struct {
	pb.UnimplementedSocialGraphServer
}

const serviceName = "graph"

var log = utils.GetLogger(serviceName)

func (s *GraphServer) Run() error {
	log.Info("Initializing gRPC server...")

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
			priceTable.UnaryInterceptor))
	}

	var dagorNode *dagor.Dagor
	if utils.GetEnvVar("dagor", false) == "true" {
		log.Info("dagor is enabled, configuring dagor interceptor")
		dagorNode = dagorinit.GetDagorNode(serviceName, true, false)
		opts = append(opts, grpc.ChainUnaryInterceptor(
			utils.ContextPropagationInterceptor(),
			counter.GetInterceptor(),
			dagorNode.UnaryInterceptorServer))
	}

	srv := grpc.NewServer(opts...)
	pb.RegisterSocialGraphServer(srv, s)

	// listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.StrToInt(utils.GetEnvVar("GraphPort", true))))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	return srv.Serve(lis)
}

func main() {
	srv := &GraphServer{}

	// initialize Redis
	populateUsersAndFollows(
		srv,
		utils.StrToInt(utils.GetEnvVar("num_of_users", true)),
		utils.StrToInt(utils.GetEnvVar("num_of_followers", true)),
	)

	log.Info("Starting server...")
	log.Error(srv.Run().Error())
}

func populateUsersAndFollows(s *GraphServer, numOfUsers, numOfFollowers int) {

	ctx := context.Background()

	for i := 0; i < numOfUsers; i++ {
		userId := fmt.Sprintf("user%d", i)

		// Insert user
		_, err := s.InsertUser(ctx, &pb.InsertUserRequest{UserId: userId})
		if err != nil {
			//log.Printf("Failed to insert user %s: %v", userId, err)
			log.Error("Failed to insert user", "userId", userId, "error", err)
		} else {
			log.Debug("Inserted user", "userId", userId)
		}
	}

	for i := 0; i < numOfUsers; i++ {
		// Follow other users
		userId := fmt.Sprintf("user%d", i)
		// if i+1 < numOfUsers {
		// each user follows the last numOfFollowers users
		if i > numOfFollowers {
			for j := 0; j < numOfFollowers; j++ {
				followeeId := fmt.Sprintf("user%d", i-j)
				_, err := s.Follow(ctx, &pb.FollowRequest{
					FollowerId: userId,
					FolloweeId: followeeId,
				})
				if err != nil {
					log.Error("Failed to follow user", "followerId", userId, "followeeId", followeeId, "error", err)
				} else {
					log.Debug("User followed user", "followerId", userId, "followeeId", followeeId)
				}
			}
		} else {
			for j := 0; j < i; j++ {
				followeeId := fmt.Sprintf("user%d", i-j)
				_, err := s.Follow(ctx, &pb.FollowRequest{
					FollowerId: userId,
					FolloweeId: followeeId,
				})
				if err != nil {
					log.Error("Failed to follow user", "followerId", userId, "followeeId", followeeId, "error", err)
				} else {
					log.Debug("User followed user", "followerId", userId, "followeeId", followeeId)
				}
			}
		}
	}
}

// InsertUser inserts a user with an empty social graph
func (s *GraphServer) InsertUser(ctx context.Context, req *pb.InsertUserRequest) (*pb.InsertUserResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	sg := utils.SGVertex{
		UserId:    req.UserId,
		Followers: []string{},
		Followees: []string{},
	}
	err := utils.SetState(ctx, req.UserId, sg)
	if err != nil {
		//config.DebugLog("Error inserting user %s: %v", req.UserId, err)
		log.Debug("Error inserting user", "userId", req.UserId, "error", err)
		return nil, err
	}
	log.Debug("Inserted user", "userId", req.UserId)
	return &pb.InsertUserResponse{}, nil
}

// GetFollowers retrieves the list of followers for a given user
func (s *GraphServer) GetFollowers(ctx context.Context, req *pb.GetFollowersRequest) (*pb.GetFollowersResponse, error) {
	utils.BusyLoop(100)
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	sg, err := utils.GetState[utils.SGVertex](ctx, req.UserId)
	if err != nil {
		log.Error("Error getting followers for user", "userId", req.UserId, "error", err)
		return nil, err
	}
	log.Debug("Retrieved followers for user", "userId", req.UserId, "followers", sg.Followers)
	return &pb.GetFollowersResponse{Followers: sg.Followers}, nil
}

// GetFollowees retrieves the list of followees for a given user
func (s *GraphServer) GetFollowees(ctx context.Context, req *pb.GetFolloweesRequest) (*pb.GetFolloweesResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	sg, err := utils.GetState[utils.SGVertex](ctx, req.UserId)
	if err != nil {
		log.Error("Error getting followees for user", "userId", req.UserId, "error", err)
		return nil, err
	}
	log.Debug("Retrieved followees for user", "userId", req.UserId, "followees", sg.Followees)
	return &pb.GetFolloweesResponse{Followees: sg.Followees}, nil
}

// Follow allows a user to follow another user
func (s *GraphServer) Follow(ctx context.Context, req *pb.FollowRequest) (*pb.FollowResponse, error) {
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	err := s.follow(ctx, req.FollowerId, req.FolloweeId)
	if err != nil {
		log.Error("Error following user", "followerId", req.FollowerId, "followeeId", req.FolloweeId, "error", err)
		return nil, err
	}
	log.Debug("User followed user", "followerId", req.FollowerId, "followeeId", req.FolloweeId)
	return &pb.FollowResponse{}, nil
}

// follow is a helper function to handle the following logic
func (s *GraphServer) follow(ctx context.Context, followerId string, followeeId string) error {
	//ctx = config.PropagateMetadata(ctx, "socialgraph")
	// Retrieve the follower's state
	sgFollower, err := utils.GetState[utils.SGVertex](ctx, followerId)
	if err != nil {
		log.Error("Error getting state for follower", "followerId", followerId, "error", err)
		sgFollower = utils.SGVertex{
			UserId:    followerId,
			Followers: []string{},
			Followees: []string{},
		}
	}
	log.Debug("Before following: follower", "followerId", followerId, "followees", sgFollower.Followees)
	// Add the followee to the follower's followees
	sgFollower.Followees = append(sgFollower.Followees, followeeId)
	err = utils.SetState(ctx, followerId, sgFollower)
	if err != nil {
		log.Error("Error setting state for follower", "followerId", followerId, "error", err)
		return err
	}
	log.Debug("After following: follower", "followerId", followerId, "followees", sgFollower.Followees)

	// Retrieve the followee's state
	sgFollowee, err := utils.GetState[utils.SGVertex](ctx, followeeId)
	if err != nil {
		log.Error("Error getting state for followee", "followeeId", followeeId, "error", err)
		sgFollowee = utils.SGVertex{
			UserId:    followeeId,
			Followers: []string{},
			Followees: []string{},
		}
	}
	log.Debug("Before following: followee", "followeeId", followeeId, "followers", sgFollowee.Followers)
	// Add the follower to the followee's followers
	sgFollowee.Followers = append(sgFollowee.Followers, followerId)
	err = utils.SetState(ctx, followeeId, sgFollowee)
	if err != nil {
		log.Error("Error setting state for followee", "followeeId", followeeId, "error", err)
		return err
	}
	log.Debug("After following: followee", "followeeId", followeeId, "followers", sgFollowee.Followers)
	return nil
}
