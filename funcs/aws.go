package funcs

import (
	"context"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sync"

	"github.com/hairyhenderson/gomplate/v3/aws"
	"github.com/hairyhenderson/gomplate/v3/conv"
)

// AWSNS - the aws namespace
// Deprecated: don't use
//nolint:golint
func AWSNS() *Funcs {
	return &Funcs{}
}

// AWSFuncs -
// Deprecated: use CreateAWSFuncs instead
func AWSFuncs(f map[string]interface{}) {
	f2 := CreateAWSFuncs(context.Background())
	for k, v := range f2 {
		f[k] = v
	}
}

// CreateAWSFuncs -
func CreateAWSFuncs(ctx context.Context) map[string]interface{} {
	f := map[string]interface{}{}

	ns := &Funcs{
		ctx:     ctx,
		awsopts: aws.GetClientOptions(),
	}

	f["aws"] = func() interface{} { return ns }

	// global aliases - for backwards compatibility
	f["ec2meta"] = ns.EC2Meta
	f["ec2dynamic"] = ns.EC2Dynamic
	f["ec2tag"] = ns.EC2Tag
	f["ec2tags"] = ns.EC2Tags
	f["ec2region"] = ns.EC2Region
	f["ec2query"] = ns.EC2Query

	return f
}

// Funcs -
type Funcs struct {
	ctx context.Context

	meta     *aws.Ec2Meta
	info     *aws.Ec2Info
	kms      *aws.KMS
	sts      *aws.STS
	ec2query *aws.Ec2Query

	metaInit     sync.Once
	infoInit     sync.Once
	kmsInit      sync.Once
	stsInit      sync.Once
	ec2QueryInit sync.Once

	awsopts aws.ClientOptions
}

// EC2Region -
func (a *Funcs) EC2Region(def ...string) (string, error) {
	a.metaInit.Do(a.initMeta)
	return a.meta.Region(def...)
}

// EC2Meta -
func (a *Funcs) EC2Meta(key string, def ...string) (string, error) {
	a.metaInit.Do(a.initMeta)
	return a.meta.Meta(key, def...)
}

// EC2Dynamic -
func (a *Funcs) EC2Dynamic(key string, def ...string) (string, error) {
	a.metaInit.Do(a.initMeta)
	return a.meta.Dynamic(key, def...)
}

// EC2Tag -
func (a *Funcs) EC2Tag(tag string, def ...string) (string, error) {
	a.infoInit.Do(a.initInfo)
	return a.info.Tag(tag, def...)
}

// EC2Tag -
func (a *Funcs) EC2Tags() (map[string]string, error) {
	a.infoInit.Do(a.initInfo)
	return a.info.Tags()
}

// KMSEncrypt -
func (a *Funcs) KMSEncrypt(keyID, plaintext interface{}) (string, error) {
	a.kmsInit.Do(a.initKMS)
	return a.kms.Encrypt(conv.ToString(keyID), conv.ToString(plaintext))
}

// KMSDecrypt -
func (a *Funcs) KMSDecrypt(ciphertext interface{}) (string, error) {
	a.kmsInit.Do(a.initKMS)
	return a.kms.Decrypt(conv.ToString(ciphertext))
}

// EC2Query -
func (a *Funcs) EC2Query(tags string) ([]*ec2.Instance, error) {
	a.ec2QueryInit.Do(a.initEC2Query)
	return a.ec2query.Query(tags)
}

// UserID - Gets the unique identifier of the calling entity. The exact value
// depends on the type of entity making the call. The values returned are those
// listed in the aws:userid column in the Principal table
// (http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_variables.html#principaltable)
// found on the Policy Variables reference page in the IAM User Guide.
func (a *Funcs) UserID() (string, error) {
	a.stsInit.Do(a.initSTS)
	return a.sts.UserID()
}

// Account - Gets the AWS account ID number of the account that owns or
// contains the calling entity.
func (a *Funcs) Account() (string, error) {
	a.stsInit.Do(a.initSTS)
	return a.sts.Account()
}

// ARN - Gets the AWS ARN associated with the calling entity
func (a *Funcs) ARN() (string, error) {
	a.stsInit.Do(a.initSTS)
	return a.sts.Arn()
}

func (a *Funcs) initMeta() {
	if a.meta == nil {
		a.meta = aws.NewEc2Meta(a.awsopts)
	}
}

func (a *Funcs) initInfo() {
	if a.info == nil {
		a.info = aws.NewEc2Info(a.awsopts)
	}
}

func (a *Funcs) initKMS() {
	if a.kms == nil {
		a.kms = aws.NewKMS(a.awsopts)
	}
}

func (a *Funcs) initSTS() {
	if a.sts == nil {
		a.sts = aws.NewSTS(a.awsopts)
	}
}

func (a *Funcs) initEC2Query() {
	if a.ec2query == nil {
		a.ec2query = aws.NewEc2Query(a.awsopts)
	}
}
