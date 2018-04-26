package ecsclient

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	//"github.com/blinkist/skipper/helpers"
)

// Ecsclient type
type Ecsclient struct {
	session      *session.Session
	svc          *ecs.ECS
	creds        *credentials.Credentials
	logger       *log.Logger
	pollInterval time.Duration
	timeout      int
}

// TaskInfo holds flattened Task Information in one struct
type TaskInfo struct {
	TaskRoleArn          *string
	TaskDefinitionArn    *string
	TaskArn              *string
	ContainerInstanceArn *string
	ContainerPort0       *int64
	Hostport0            *int64
	Proto0               *string
	AwsLogGroup          *string
	Ec2InstanceId        *string
	IpAddress            *string
	Cpu0                 *int64
	Softmem0             *int64
	Hardmem0             *int64
}

var instance *Ecsclient
var once sync.Once

// New Constructor
func New() *Ecsclient {
	sess := session.New()
	svc := ecs.New(sess)
	logger := log.New(os.Stderr, " - ", log.LstdFlags)

	return &Ecsclient{
		svc:          svc,
		pollInterval: time.Second * 5,
		logger:       logger,
		session:      sess,
		timeout:      300,
	}
}

// Singleton method
func GetInstance() *Ecsclient {
	once.Do(func() {
		instance = New()
	})
	return instance
}

// Scale Service to desired Count
func (c *Ecsclient) ScaleService(cluster string, service string, desiredCount int) (*ecs.Service, error) {
	input := &ecs.UpdateServiceInput{}
	input.SetCluster(cluster)
	input.SetService(service)
	input.SetDesiredCount(int64(desiredCount))
	output, err := c.svc.UpdateService(input)
	if err != nil {
		return nil, err
	}
	return output.Service, nil
}

// ListServices of a cluster
func (c *Ecsclient) ListServices(cluster *string) ([]string, error) {
	params := &ecs.ListServicesInput{Cluster: aws.String(*cluster)}
	result := make([]*string, 0)

	err := c.svc.ListServicesPages(params, func(services *ecs.ListServicesOutput, lastPage bool) bool {
		result = append(result, services.ServiceArns...)
		return !lastPage
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, len(result))
	for i, s := range result {
		out[i] = path.Base(*s)
	}
	return out, nil
}

// Returns ECS Service of Cluster
func (c *Ecsclient) FindService(cluster *string, service *string) (*ecs.Service, error) {
	result, err := c.svc.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(*cluster),
		Services: []*string{aws.String(*service)},
	})
	if err != nil {
		return nil, err
	}
	if len(result.Services) == 1 {
		return result.Services[0], nil
	}
	return nil, fmt.Errorf("did not find one (%d) services matching name %s in %s cluster, unable to continue", len(result.Services), service, *cluster)
}

// Start Task on Container Instance
func (c *Ecsclient) StartTaskOnContainerInstance(cluster *string, taskdefinition *string, container *string, rolearn *string, startedBy *string) (*ecs.StartTaskOutput, error) {

	sti := &ecs.StartTaskInput{
		StartedBy:          aws.String(*startedBy),
		TaskDefinition:     aws.String(*taskdefinition),
		Cluster:            aws.String(*cluster),
		ContainerInstances: []*string{container},
		Overrides: &ecs.TaskOverride{
			TaskRoleArn: aws.String(*rolearn),
		},
	}
	sto, err := c.svc.StartTask(sti)
	return sto, err
}

// Stop Task on cluster
func (c *Ecsclient) StopTask(cluster *string, taskarn *string) (bool, error) {
	_, err := c.svc.StopTask(&ecs.StopTaskInput{
		Cluster: aws.String(*cluster),
		Task:    taskarn,
	})

	if err != nil {
		return false, err
	}
	return true, nil
}

// Return available cluster names
func (c *Ecsclient) GetClusterNames() ([]string, error) {
	retCluster := make([]string, 0)

	listclustersOutput, err := c.svc.ListClusters(&ecs.ListClustersInput{})

	if listclustersOutput.NextToken != nil {
		fmt.Println("Pagination not implemented")
		os.Exit(1)
	}

	if err != nil {
		return nil, err
	}

	for _, arn := range listclustersOutput.ClusterArns {
		tmp := strings.Split(*arn, "/")
		retCluster = append(retCluster, tmp[len(tmp)-1])
	}

	return retCluster, nil
}

// Input for RegisterTaskDefinition
type RegisterTaskDefinitionInput struct {
	Image                *string
	Tag                  *string
	ContainerInstanceArn *string
	Cpu                  *int64
	Hostport0            *int64
	Softmem              *int64
	Hardmem              *int64
	Envvars              *string
	TargetGroup          *string
	Changes              *map[string]string
	Unsets               *map[string]struct{}
}

// RegisterTaskDefinition updates the existing task definition's image.
func (c *Ecsclient) RegisterTaskDefinition(task *string, rdi *RegisterTaskDefinitionInput) (string, error) {

	defs, err := c.GetContainerDefinitions(task)
	if err != nil {
		return "", err
	}

	for i, d := range defs {

		if rdi.Cpu != nil {
			d.Cpu = rdi.Cpu
		}

		if rdi.Tag != nil && rdi.Image != nil && strings.HasPrefix(*d.Image, *rdi.Image) {
			i := fmt.Sprintf("%s:%s", *rdi.Image, *rdi.Tag)
			d.Image = &i
		}

		if rdi.Cpu != nil {
			d.Cpu = rdi.Cpu
		}

		if rdi.Hardmem != nil {
			d.Memory = rdi.Hardmem
		}

		if rdi.Softmem != nil {
			d.MemoryReservation = rdi.Softmem
		}

		var hasChanged bool
		if rdi.Changes != nil {

			envvars := make([]*ecs.KeyValuePair, 0, len(d.Environment))

			currentChanges := make(map[string]string, len(*rdi.Changes))
			for k, v := range *rdi.Changes {
				currentChanges[k] = v
			}

			for _, envvar := range d.Environment {
				newValue, ok := currentChanges[*envvar.Name]
				if ok {
					if !strings.EqualFold(*envvar.Value, newValue) {
						envvar.SetValue(newValue)
						hasChanged = true
					}

					delete(currentChanges, *envvar.Name)
				}

				if _, ok = (*rdi.Unsets)[*envvar.Name]; ok {
					hasChanged = true
					continue
				}

				envvars = append(envvars, envvar)
			}

			if len(currentChanges) > 0 {
				// Some fields that didn't exist need to be add
				hasChanged = true
			}

			for name, value := range currentChanges {
				envvars = append(envvars, &ecs.KeyValuePair{
					Name:  aws.String(name),
					Value: aws.String(value),
				})
			}
			defs[i].Environment = envvars
			fmt.Println(hasChanged)
		}
	}

	taskRoleArn, err := c.GetTaskRoleArn(task)
	fmt.Println(taskRoleArn)
	if err != nil {
		return "", err
	}

	placementConstraints, err := c.GetTaskPlacementConstraints(task)
	if err != nil {
		return "", err
	}

	if rdi.TargetGroup != nil {
		targetGroupChanged := false
		expression := fmt.Sprintf("attribute:group == %s", *rdi.TargetGroup)
		for i, constraint := range placementConstraints {
			if (*constraint.Expression)[0:16] == "attribute:group" {

				placementConstraints[i].Expression = &expression
				targetGroupChanged = true
			}
		}

		if targetGroupChanged == false {
			placementConstraints = append(placementConstraints,
				&ecs.TaskDefinitionPlacementConstraint{
					Expression: aws.String(expression),
					Type:       aws.String("memberOf"),
				})
		}
	}

	volumes, err := c.GetTaskVolumes(task)
	if err != nil {
		return "", err
	}
	networkMode, err := c.GetContainerNetworkMode(task)
	if err != nil {
		return "", err
	}

	input := &ecs.RegisterTaskDefinitionInput{
		Family:               task,
		ContainerDefinitions: defs,
		TaskRoleArn:          taskRoleArn,
		PlacementConstraints: placementConstraints,
		Volumes:              volumes,
		NetworkMode:          networkMode,
	}

	resp, err := c.svc.RegisterTaskDefinition(input)

	if err != nil {
		return "", err
	}

	return *resp.TaskDefinition.TaskDefinitionArn, nil
}

// Wait waits for the service to finish being updated.
func (c *Ecsclient) Wait(cluster, service, arn *string) error {
	t := time.NewTicker(c.pollInterval)
	start := time.Now()
	for {
		select {
		case <-t.C:
			s, err := c.GetDeployment(cluster, service, arn)
			if err != nil {
				return err
			}
			c.logger.Printf("[info] --> desired: %d, pending: %d, running: %dm, elapsed secs: %s", *s.DesiredCount, *s.PendingCount, *s.RunningCount, time.Since(start))
			if *s.RunningCount == *s.DesiredCount {
				return nil
			}
			if time.Now().Unix() > start.Add(time.Second*time.Duration(c.timeout)).Unix() {
				c.logger.Printf("[info] --> desired:%d - %d", time.Now().Unix(), start.Add(time.Second*time.Duration(c.timeout)).Unix())
				return errors.New("waiting timed out")
			}
		}
	}
}

// GetDeployment gets the deployment for the arn.
func (c *Ecsclient) GetDeployment(cluster, service, arn *string) (*ecs.Deployment, error) {
	input := &ecs.DescribeServicesInput{
		Cluster:  cluster,
		Services: []*string{service},
	}

	output, err := c.svc.DescribeServices(input)

	if err != nil {
		return nil, err
	}
	ds := output.Services[0].Deployments
	for _, d := range ds {
		if *d.TaskDefinition == *arn {
			return d, nil
		}
	}
	return nil, nil
}

// GetContainerDefinitions get container definitions of the service.
func (c *Ecsclient) GetContainerImage(cluster *string, service *string) *string {
	tcs, _ := c.GetContainerInstances(cluster, service)
	a, _ := c.GetContainerDefinitions(tcs[0].TaskDefinitionArn)
	return a[0].Image
}

// GetContainerDefinitions get container definitions of the service.
func (c *Ecsclient) GetContainerDefinitions(task *string) ([]*ecs.ContainerDefinition, error) {

	output, err := c.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task,
	})
	if err != nil {
		return nil, err
	}
	return output.TaskDefinition.ContainerDefinitions, nil
}

// GetContainerDefinitions get container definitions of the service.
func (c *Ecsclient) GetContainerNetworkMode(task *string) (*string, error) {

	output, err := c.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task,
	})
	if err != nil {
		return nil, err
	}
	return output.TaskDefinition.NetworkMode, nil
}

// GetContainerDefinitions get container definitions of the service.
func (c *Ecsclient) GetTaskRoleArn(task *string) (*string, error) {

	output, err := c.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task,
	})

	if err != nil {
		return nil, err
	}
	return output.TaskDefinition.TaskRoleArn, nil
}

// GetContainerDefinitions get container definitions of the service.
func (c *Ecsclient) GetTaskVolumes(task *string) ([]*ecs.Volume, error) {

	output, err := c.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task,
	})

	if err != nil {
		return nil, err
	}
	return output.TaskDefinition.Volumes, nil
}

// GetContainerDefinitions get container definitions of the service.
func (c *Ecsclient) GetTaskPlacementConstraints(task *string) ([]*ecs.TaskDefinitionPlacementConstraint, error) {

	output, err := c.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task,
	})

	if err != nil {
		return nil, err
	}
	return output.TaskDefinition.PlacementConstraints, nil
}

// Restart service
func (c *Ecsclient) RestartService(cluster *string, service *string) {
	c.UpdateService(cluster, service, nil, nil, true, nil, nil, nil)
}

// UpdateService
func (c *Ecsclient) UpdateService(cluster, service *string, image *string, count *int64, argWait bool, changes *map[string]string, unsets *map[string]struct{}, targetGroup *string) {

	rtd := &RegisterTaskDefinitionInput{
		Image:       image,
		Tag:         nil,
		Cpu:         nil,
		Hardmem:     nil,
		Softmem:     nil,
		TargetGroup: targetGroup,
		Changes:     changes,
		Unsets:      unsets,
	}

	task := service

	arn, err := c.RegisterTaskDefinition(task, rtd)

	if err != nil {
		fmt.Printf("[error] register task definition: %s\n", err)
		return
	}

	err = c.UpdateServiceWithTaskDefinition(cluster, service, count, &arn)

	if err != nil {
		fmt.Printf("[error] update service: %s\n", err)
	}

	if argWait == true {
		err := c.Wait(cluster, service, &arn)

		if err != nil {
			fmt.Printf("[error] wait: %s\n", err)
			return
		}
	}
}

// Wait for task running
func (c *Ecsclient) WaitForTaskRunning(cluster *string, taskArn *string) error {
	params := &ecs.DescribeTasksInput{
		Cluster: aws.String(*cluster),
		Tasks:   []*string{aws.String(*taskArn)},
	}
	return c.svc.WaitUntilTasksRunning(params)
}

// UpdateServiceWithTaskDefinition updates the service to use the new task definition.
func (c *Ecsclient) UpdateServiceWithTaskDefinition(cluster *string, service *string, count *int64, arn *string) error {
	input := &ecs.UpdateServiceInput{
		Cluster: cluster,
		Service: service,
	}
	if count != nil {
		input.DesiredCount = count
	}
	if arn != nil {
		input.TaskDefinition = arn
	}
	_, err := c.svc.UpdateService(input)
	return err
}

// Get cluster tasks with definition X
func (c *Ecsclient) GetClusterTasksWithDefinition(cluster *string, taskdefinition *string) ([]*ecs.Task, error) {
	tasks, err := c.GetClusterTasks(cluster)

	if err != nil {
		return nil, err
	}

	j := 0
	for i, _ := range tasks {
		if *tasks[i].TaskDefinitionArn == *taskdefinition {
			tasks[j] = tasks[i]
			j++
		}
	}
	tasks = tasks[:j]

	return tasks, nil
}

// Get cluster tasks
func (c *Ecsclient) GetClusterTasks(cluster *string) ([]*ecs.Task, error) {
	input := &ecs.ListTasksInput{}
	input.SetCluster(*cluster)

	result, err := c.svc.ListTasks(input)
	if err != nil {
		return nil, err
	}
	if result.NextToken != nil {
		fmt.Println("Pagination not implemented")
		os.Exit(1)
	}
	input2 := &ecs.DescribeTasksInput{}
	input2.SetCluster(*cluster)
	input2.SetTasks(result.TaskArns)
	result2, err := c.svc.DescribeTasks(input2)

	return result2.Tasks, nil
}

// GetContainerInstances returns the container instances of a cluster
func (c *Ecsclient) GetContainerInstances(cluster *string, service *string) ([]*TaskInfo, error) {

	input := &ecs.ListTasksInput{}
	input.SetCluster(*cluster)
	input.SetServiceName(*service)

	result, err := c.svc.ListTasks(input)
	if err != nil {
		return nil, err
	}

	input2 := &ecs.DescribeTasksInput{}
	input2.SetCluster(*cluster)
	input2.SetTasks(result.TaskArns)

	result2, err := c.svc.DescribeTasks(input2)
	if err != nil {
		return nil, err
	}
	instances := make([]*string, 0)

	var mytaskinstances []*TaskInfo

	for _, t := range result2.Tasks {
		//fmt.Println(t.TaskDefinitionArn)
		//for a, b := range result2.Tasks {
		var awsloggroup *string

		defs, _ := c.GetContainerDefinitions(t.TaskDefinitionArn)

		//fmt.Println(defs[0].LogConfiguration.Options)
		for k, v := range defs[0].LogConfiguration.Options {
			//fmt.Println(k)
			if k == "awslogs-group" {
				awsloggroup = v
			}
		}
		if *t.LastStatus == "RUNNING" {
			tc := TaskInfo{TaskDefinitionArn: t.TaskDefinitionArn,
				Cpu0:                 defs[0].Cpu,
				Hardmem0:             defs[0].Memory,
				Softmem0:             defs[0].MemoryReservation,
				TaskArn:              t.TaskArn,
				AwsLogGroup:          awsloggroup,
				ContainerInstanceArn: t.ContainerInstanceArn,
				ContainerPort0:       t.Containers[0].NetworkBindings[0].ContainerPort,
				Hostport0:            t.Containers[0].NetworkBindings[0].HostPort,
				Proto0:               t.Containers[0].NetworkBindings[0].Protocol}

			mytaskinstances = append(mytaskinstances, &tc)
			instances = append(instances, t.ContainerInstanceArn)
		}
	}

	instanceInput := &ecs.DescribeContainerInstancesInput{}
	instanceInput.SetCluster(*cluster)
	instanceInput.SetContainerInstances(instances)
	result3, err := c.svc.DescribeContainerInstances(instanceInput)
	if err != nil {
		return nil, err
	}

	ec2svc := ec2.New(c.session, &aws.Config{Credentials: c.creds})

	iinput := &ec2.DescribeInstancesInput{}
	ec2instances := make([]*string, len(result3.ContainerInstances))

	for i, ci := range result3.ContainerInstances {
		for _, ti := range mytaskinstances {
			if *ti.ContainerInstanceArn == *ci.ContainerInstanceArn {
				ti.Ec2InstanceId = ci.Ec2InstanceId
			}
		}
		ec2instances[i] = ci.Ec2InstanceId
	}

	iinput.SetInstanceIds(ec2instances)
	result4, err := ec2svc.DescribeInstances(iinput)

	if err != nil {
		return nil, err
	}

	dnsNames := make([]string, len(result4.Reservations))
	for i, r := range result4.Reservations {
		dnsNames[i] = *r.Instances[0].PrivateIpAddress
		for _, ti := range mytaskinstances {

			if *ti.Ec2InstanceId == *r.Instances[0].InstanceId {
				ti.IpAddress = r.Instances[0].PrivateIpAddress
			}
		}
	}
	return mytaskinstances, nil
}

// Describe container instances
func (c *Ecsclient) DescribeContainerInstances(cluster *string, instances []*ec2.Instance) ([]*ecs.ContainerInstance, error) {

	ec2instances := make([]*string, 0)
	for _, inst := range instances {
		ec2instances = append(ec2instances, inst.InstanceId)
	}

	listInput := &ecs.ListContainerInstancesInput{}
	listInput.SetCluster(*cluster)
	res, err := c.svc.ListContainerInstances(listInput)
	if err != nil {
		return nil, err
	}

	instanceInput := &ecs.DescribeContainerInstancesInput{}
	instanceInput.SetCluster(*cluster)
	instanceInput.SetContainerInstances(res.ContainerInstanceArns)

	result3, err := c.svc.DescribeContainerInstances(instanceInput)
	if err != nil {
		return nil, err
	}

	out := make([]*ecs.ContainerInstance, 0)
	for _, ci := range result3.ContainerInstances {
		for _, inst := range instances {
			if *ci.Ec2InstanceId == *inst.InstanceId {
				out = append(out, ci)
			}
		}
	}

	return out, nil
}

// check if string exists in slice
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Get instance id for container arn
func (c *Ecsclient) GetInstanceIDForContainerArn(cluster *string, containerinstancearn *string) (*string, error) {
	instanceInput := &ecs.DescribeContainerInstancesInput{}
	instanceInput.SetCluster(*cluster)
	instanceInput.SetContainerInstances([]*string{containerinstancearn})

	result3, err := c.svc.DescribeContainerInstances(instanceInput)
	if err != nil {
		return nil, err
	}

	if len(result3.ContainerInstances) != 1 {
		return nil, fmt.Errorf("Could not get the EC2Instance for this ARN =")
	}

	instanceId := result3.ContainerInstances[0].Ec2InstanceId

	return instanceId, nil
}

// Get tasksarns for service
func (c *Ecsclient) GetTaskArnsForService(cluster *string, service *string) ([]*string, error) {
	tcs, err := c.GetContainerInstances(cluster, service)

	if err != nil {
		return nil, err
	}

	taskarns := make([]*string, len(tcs))

	for i, ti := range tcs {
		taskarns[i] = ti.TaskArn
	}
	return taskarns, nil
}
