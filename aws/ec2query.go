package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strings"
)

type Ec2Query struct {
	ec2 *ec2.EC2
}

func NewEc2Query(_ ClientOptions) *Ec2Query {
	svc := ec2.New(SDKSession())
	return &Ec2Query{ec2: svc}
}

func (e *Ec2Query) Query(filter string) ([]*ec2.Instance, error) {
	filters := make([]*ec2.Filter, 0)
	splitFilter := strings.Split(filter, ",")
	for _, str := range splitFilter {
		keyValue := strings.Split(str, "=")
		filters = append(filters, &ec2.Filter{
			Name: aws.String(keyValue[0]),
			Values: []*string{
				aws.String(keyValue[1]),
			},
		})
	}
	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	result, err := e.ec2.DescribeInstances(input)
	if err != nil {
		return nil, err
	}
	instances := make([]*ec2.Instance, 0)
MoreReservations:
	for _, r := range result.Reservations {
		instances = append(instances, r.Instances...)
	}
	if result.NextToken != nil {
		input = &ec2.DescribeInstancesInput{
			NextToken: result.NextToken,
		}
		result, err = e.ec2.DescribeInstances(input)
		if err != nil {
			return nil, err
		}
		goto MoreReservations
	}
	return instances, nil
}
