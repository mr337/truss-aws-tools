package main

import (
	"github.com/trussworks/truss-aws-tools/internal/aws/session"
	"github.com/trussworks/truss-aws-tools/amiclean"
	"github.com/aws/aws-sdk-go/service/ec2"
	flag "github.com/jessevdk/go-flags"
	"go.uber.org/zap"

	"log"
	"time"
)

// The Options struct describes the command line options available.
type Options struct {
	DryRun bool `short:"n" long:"dryrun" description:"Run in dryrun mode
	(ie, do not actually purge AMIs)."`
	RetentionDays int `long:"days" default:"30" description:"Age of AMI in
	days before it is a candidate for removal."`
	Branch string `short:"b" long:"branch" description:"Branch to purge.
	Preface with ! to purge all branches *but* this one (eg, !master would
	purge all AMIs not from the master branch)."`
	Profile string `short:"p" long:"profile" env:"PROFILE" required:"false"
	description:"The AWS profile to use."`
	Region string `short:"r" long:"region" env:"REGION" required:"false"
	description:"The AWS region to use."`
}

var options Options
var logger *zap.Logger

// This function is for establishing our session with AWS.
func makeEC2Client(region, profile string) *ec2.EC2 {
	sess := session.MustMakeSession(region, profile)
	ec2Client := ec2.New(sess)
	return ec2Client
}

func cleanImages() {
	now := Time.Now().UTC()
	a := amiclean.AMIClean{
		Branch: options.Branch,
		DryRun: options.DryRun,
		ExpirationDate: now.AddDate(0, 0, -int(options.RetentionDays)),
		Logger: logger,
		EC2Client: makeEC2Client(options.Region, options.Profile),
	}

	availableImages, err := a.GetImages()
	if err != nil {
		logger.Fatal("unable to get list of available images",
			zap.Error(err)
		)
	}

	purgeList := a.FindImagesToPurge(availableImages)

	amiIdsToPurge, snapshotIdsToPurge := a.GetIdsToProcess(purgeList)

	err = a.DeregisterImageList(amiIdsToPurge)
	if err != nil {
		logger.Fatal("unable to deregister AMIs",
			zap.Error(err)
		)
	}

	err = a.DeleteSnapshotList(snapshotIdsToPurge)
	if err != nil {
		logger.Fatal("unable to delete snapshots",
			zap.Error(err)
		)
	}

}

func main() {
	// First, parse out our command line options:
	parser := flag.NewParser(&options, flag.Default)
	_, err := parser.Parse()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the zap logger:
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}

	// And now we just call cleanImages to actually do the work.
	cleanImages()

}
