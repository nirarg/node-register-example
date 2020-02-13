package noderegister

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/go-logr/logr"

	appv1alpha1 "github.com/example-inc/app-operator/pkg/apis/app/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_noderegister")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new NodeRegister Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNodeRegister{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("noderegister-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NodeRegister
	err = c.Watch(&source.Kind{Type: &appv1alpha1.NodeRegister{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner NodeRegister
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1alpha1.NodeRegister{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNodeRegister implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNodeRegister{}

// ReconcileNodeRegister reconciles a NodeRegister object
type ReconcileNodeRegister struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a NodeRegister object and makes changes based on the state read
// and what is in the NodeRegister.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNodeRegister) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NodeRegister")

	reqLogger.Info("Fetch the NodeRegister instance")
	nodeRegister := &appv1alpha1.NodeRegister{}
	reqLogger.Info("Get nodeRegister instance")
	err := r.client.Get(context.TODO(), request.NamespacedName, nodeRegister)
	if err != nil {
		reqLogger.Info("nodeRegister instance not found. Ignoring since object must be deleted")
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get nodeRegister instance")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Check if the deployment already exists, if not create a new one")
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: nodeRegister.Name, Namespace: nodeRegister.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		reqLogger.Info("Deployment is not found")
		dep := r.deploymentForNodeRegister(nodeRegister, reqLogger)
		reqLogger.Info("Set NodeRegister instance as the owner and controller of the deploy object")
		if err := controllerutil.SetControllerReference(nodeRegister, dep, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
		reqLogger.Info("Deployment created successfully - return and requeue")
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Ensure the deployment size is the same as the spec")
	size := nodeRegister.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return reconcile.Result{}, err
		}
		reqLogger.Info("Spec updated - return and requeue")
		return reconcile.Result{Requeue: true}, nil
	}

	reqLogger.Info("Update the NodeRegister status with the pod names")
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(nodeRegister.Namespace),
		client.MatchingLabels(labelsForNodeRegister(nodeRegister.Name)),
	}
	if err = r.client.List(context.TODO(), podList, listOpts...); err != nil {
		reqLogger.Error(err, "Failed to list pods", "nodeRegister.Namespace", nodeRegister.Namespace, "nodeRegister.Name", nodeRegister.Name)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	reqLogger.Info("Update status.Nodes if needed")
	if !reflect.DeepEqual(podNames, nodeRegister.Status.Nodes) {
		nodeRegister.Status.Nodes = podNames
		err := r.client.Status().Update(context.TODO(), nodeRegister)
		if err != nil {
			reqLogger.Error(err, "Failed to update nodeRegister status")
			return reconcile.Result{}, err
		}
	}

	reqLogger.Info("Define a new service object")
	service := newServiceForCR(nodeRegister)

	reqLogger.Info("Check if the service already exists")
	foundService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service", "service.Namespace", service.Namespace, "service.Name", service.Name)
		err = r.client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{}, err
		}

		reqLogger.Info("Set NodeRegister instance as the owner and controller of the service")
		if err := controllerutil.SetControllerReference(nodeRegister, service, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		reqLogger.Info("Service created successfully - don't requeue")
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	reqLogger.Info("Service already exists - don't requeue")

	return reconcile.Result{}, nil
}

// deploymentForNodeRegister returns a nodeRegister Deployment object
func (r *ReconcileNodeRegister) deploymentForNodeRegister(nr *appv1alpha1.NodeRegister, reqLogger logr.Logger) *appsv1.Deployment {
	ls := labelsForNodeRegister(nr.Name)
	replicas := nr.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nr.Name,
			Namespace: nr.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           "quay.io/nargaman/node-register-server:v0.0.1",
						Name:            nr.Name + "-pod",
						ImagePullPolicy: "IfNotPresent",
						Ports: []corev1.ContainerPort{{
							ContainerPort: nr.Spec.TargetPort,
						}},
					}},
				},
			},
		},
	}
	return dep
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newServiceForCR(nr *appv1alpha1.NodeRegister) *corev1.Service {
	ls := labelsForNodeRegister(nr.Name)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nr.Name + "-service",
			Namespace: nr.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ServiceSpec{
			Type: "NodePort",
			Ports: []corev1.ServicePort{
				{
					Port:       nr.Spec.Port,
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: nr.Spec.TargetPort},
				},
			},
			Selector: ls,
		},
	}
}

// labelsForMemcached returns the labels for selecting the resources
// belonging to the given memcached CR name.
func labelsForNodeRegister(name string) map[string]string {
	return map[string]string{"app": name + "-pod", "noderegister_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
