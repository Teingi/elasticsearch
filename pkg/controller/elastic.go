package controller

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/appscode/log"
	tapi "github.com/k8sdb/apimachinery/api"
	amc "github.com/k8sdb/apimachinery/pkg/controller"
	"github.com/k8sdb/apimachinery/pkg/eventer"
	kapi "k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

func (c *Controller) create(elastic *tapi.Elastic) error {
	var err error
	if elastic, err = c.ExtClient.Elastics(elastic.Namespace).Get(elastic.Name); err != nil {
		return err
	}

	t := unversioned.Now()
	elastic.Status.CreationTime = &t
	elastic.Status.Phase = tapi.DatabasePhaseCreating
	if _, err = c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToUpdate,
			`Fail to update Elastic: "%v". Reason: %v`,
			elastic.Name,
			err,
		)
		log.Errorln(err)
	}

	if err := c.validateElastic(elastic); err != nil {
		c.eventRecorder.Event(elastic, kapi.EventTypeWarning, eventer.EventReasonInvalid, err.Error())
		return err
	}
	// Event for successful validation
	c.eventRecorder.Event(
		elastic,
		kapi.EventTypeNormal,
		eventer.EventReasonSuccessfulValidate,
		"Successfully validate Elastic",
	)

	// Check DormantDatabase
	if err := c.findDormantDatabase(elastic); err != nil {
		return err
	}

	// Event for notification that kubernetes objects are creating
	c.eventRecorder.Event(elastic, kapi.EventTypeNormal, eventer.EventReasonCreating, "Creating Kubernetes objects")

	// create Governing Service
	governingService := c.opt.GoverningService
	if err := c.CreateGoverningService(governingService, elastic.Namespace); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			`Failed to create ServiceAccount: "%v". Reason: %v`,
			governingService,
			err,
		)
		return err
	}

	// ensure database Service
	if err := c.ensureService(elastic); err != nil {
		return err
	}

	// ensure database StatefulSet
	if err := c.ensureStatefulSet(elastic); err != nil {
		return err
	}

	c.eventRecorder.Event(
		elastic,
		kapi.EventTypeNormal,
		eventer.EventReasonSuccessfulCreate,
		"Successfully created Elastic",
	)

	// Ensure Schedule backup
	c.ensureBackupScheduler(elastic)

	if elastic.Spec.Monitor != nil {
		if err := c.addMonitor(elastic); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToAddMonitor,
				"Failed to add monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		}
		c.eventRecorder.Event(
			elastic,
			kapi.EventTypeNormal,
			eventer.EventReasonSuccessfulMonitorAdd,
			"Successfully added monitoring system.",
		)
	}
	return nil
}

func (c *Controller) findDormantDatabase(elastic *tapi.Elastic) error {
	// Check if DormantDatabase exists or not
	dormantDb, err := c.ExtClient.DormantDatabases(elastic.Namespace).Get(elastic.Name)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToGet,
				`Fail to get DormantDatabase: "%v". Reason: %v`,
				elastic.Name,
				err,
			)
			return err
		}
	} else {
		var message string
		if dormantDb.Labels[amc.LabelDatabaseKind] != tapi.ResourceKindElastic {
			message = fmt.Sprintf(`Invalid Elastic: "%v". Exists DormantDatabase "%v" of different Kind`,
				elastic.Name, dormantDb.Name)
		} else {
			message = fmt.Sprintf(`Recover from DormantDatabase: "%v"`, dormantDb.Name)
		}
		c.eventRecorder.Event(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			message,
		)
		return errors.New(message)
	}
	return nil
}

func (c *Controller) ensureService(elastic *tapi.Elastic) error {
	// Check if service name exists
	found, err := c.findService(elastic.Name, elastic.Namespace)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	// create database Service
	if err := c.createService(elastic.Name, elastic.Namespace); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			"Failed to create Service. Reason: %v",
			err,
		)
		return err
	}
	return nil
}

func (c *Controller) ensureStatefulSet(elastic *tapi.Elastic) error {
	found, err := c.findStatefulSet(elastic)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	// Create statefulSet for Elastic database
	statefulSet, err := c.createStatefulSet(elastic)
	if err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			"Failed to create StatefulSet. Reason: %v",
			err,
		)
		return err
	}

	// Check StatefulSet Pod status
	if elastic.Spec.Replicas > 0 {
		if err := c.CheckStatefulSetPodStatus(statefulSet, durationCheckStatefulSet); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToStart,
				"Failed to create StatefulSet. Reason: %v",
				err,
			)
			return err
		} else {
			c.eventRecorder.Event(
				elastic,
				kapi.EventTypeNormal,
				eventer.EventReasonSuccessfulCreate,
				"Successfully created StatefulSet",
			)
		}
	}

	if elastic.Spec.Init != nil && elastic.Spec.Init.SnapshotSource != nil {
		if elastic, err = c.ExtClient.Elastics(elastic.Namespace).Get(elastic.Name); err != nil {
			return err
		}

		elastic.Status.Phase = tapi.DatabasePhaseInitializing
		if _, err = c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToUpdate,
				`Fail to update Elastic: "%v". Reason: %v`,
				elastic.Name,
				err,
			)
			log.Errorln(err)
		}

		if err := c.initialize(elastic); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToInitialize,
				"Failed to initialize. Reason: %v",
				err,
			)
		}
	}

	if elastic, err = c.ExtClient.Elastics(elastic.Namespace).Get(elastic.Name); err != nil {
		return err
	}

	elastic.Status.Phase = tapi.DatabasePhaseRunning
	if _, err = c.ExtClient.Elastics(elastic.Namespace).Update(elastic); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToUpdate,
			`Failed to update Elastic: "%v". Reason: %v`,
			elastic.Name,
			err,
		)
		log.Errorln(err)
	}

	return nil
}

func (c *Controller) ensureBackupScheduler(elastic *tapi.Elastic) {
	// Setup Schedule backup
	if elastic.Spec.BackupSchedule != nil {
		err := c.cronController.ScheduleBackup(elastic, elastic.ObjectMeta, elastic.Spec.BackupSchedule)
		if err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToSchedule,
				"Failed to schedule snapshot. Reason: %v",
				err,
			)
			log.Errorln(err)
		}
	} else {
		c.cronController.StopBackupScheduling(elastic.ObjectMeta)
	}
}

const (
	durationCheckRestoreJob = time.Minute * 30
)

func (c *Controller) initialize(elastic *tapi.Elastic) error {
	snapshotSource := elastic.Spec.Init.SnapshotSource
	// Event for notification that kubernetes objects are creating
	c.eventRecorder.Eventf(
		elastic,
		kapi.EventTypeNormal,
		eventer.EventReasonInitializing,
		`Initializing from Snapshot: "%v"`,
		snapshotSource.Name,
	)

	namespace := snapshotSource.Namespace
	if namespace == "" {
		namespace = elastic.Namespace
	}
	snapshot, err := c.ExtClient.Snapshots(namespace).Get(snapshotSource.Name)
	if err != nil {
		return err
	}

	job, err := c.createRestoreJob(elastic, snapshot)
	if err != nil {
		return err
	}

	jobSuccess := c.CheckDatabaseRestoreJob(job, elastic, c.eventRecorder, durationCheckRestoreJob)
	if jobSuccess {
		c.eventRecorder.Event(
			elastic,
			kapi.EventTypeNormal,
			eventer.EventReasonSuccessfulInitialize,
			"Successfully completed initialization",
		)
	} else {
		c.eventRecorder.Event(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToInitialize,
			"Failed to complete initialization",
		)
	}
	return nil
}

func (c *Controller) pause(elastic *tapi.Elastic) error {
	c.eventRecorder.Event(elastic, kapi.EventTypeNormal, eventer.EventReasonPausing, "Pausing Elastic")

	if elastic.Spec.DoNotPause {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToPause,
			`Elastic "%v" is locked.`,
			elastic.Name,
		)

		if err := c.reCreateElastic(elastic); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToCreate,
				`Failed to recreate Elastic: "%v". Reason: %v`,
				elastic.Name,
				err,
			)
			return err
		}
		return nil
	}

	if _, err := c.createDormantDatabase(elastic); err != nil {
		c.eventRecorder.Eventf(
			elastic,
			kapi.EventTypeWarning,
			eventer.EventReasonFailedToCreate,
			`Failed to create DormantDatabase: "%v". Reason: %v`,
			elastic.Name,
			err,
		)
		return err
	}
	c.eventRecorder.Eventf(
		elastic,
		kapi.EventTypeNormal,
		eventer.EventReasonSuccessfulCreate,
		`Successfully created DormantDatabase: "%v"`,
		elastic.Name,
	)

	c.cronController.StopBackupScheduling(elastic.ObjectMeta)

	if elastic.Spec.Monitor != nil {
		if err := c.deleteMonitor(elastic); err != nil {
			c.eventRecorder.Eventf(
				elastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToDeleteMonitor,
				"Failed to delete monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		}
		c.eventRecorder.Event(
			elastic,
			kapi.EventTypeNormal,
			eventer.EventReasonSuccessfulMonitorDelete,
			"Successfully deleted monitoring system.",
		)
	}
	return nil
}

func (c *Controller) update(oldElastic, updatedElastic *tapi.Elastic) error {

	if err := c.validateElastic(updatedElastic); err != nil {
		c.eventRecorder.Event(updatedElastic, kapi.EventTypeWarning, eventer.EventReasonInvalid, err.Error())
		return err
	}
	// Event for successful validation
	c.eventRecorder.Event(
		updatedElastic,
		kapi.EventTypeNormal,
		eventer.EventReasonSuccessfulValidate,
		"Successfully validate Elastic",
	)

	// Check DormantDatabase
	if err := c.findDormantDatabase(updatedElastic); err != nil {
		return err
	}

	if err := c.ensureService(updatedElastic); err != nil {
		return err
	}
	if err := c.ensureStatefulSet(updatedElastic); err != nil {
		return err
	}

	if (updatedElastic.Spec.Replicas != oldElastic.Spec.Replicas) && updatedElastic.Spec.Replicas >= 0 {
		statefulSetName := getStatefulSetName(updatedElastic.Name)
		statefulSet, err := c.Client.Apps().StatefulSets(updatedElastic.Namespace).Get(statefulSetName)
		if err != nil {
			c.eventRecorder.Eventf(
				updatedElastic,
				kapi.EventTypeNormal,
				eventer.EventReasonFailedToGet,
				`Failed to get StatefulSet: "%v". Reason: %v`,
				statefulSetName,
				err,
			)
			return err
		}
		statefulSet.Spec.Replicas = updatedElastic.Spec.Replicas
		if _, err := c.Client.Apps().StatefulSets(statefulSet.Namespace).Update(statefulSet); err != nil {
			c.eventRecorder.Eventf(
				updatedElastic,
				kapi.EventTypeNormal,
				eventer.EventReasonFailedToUpdate,
				`Failed to update StatefulSet: "%v". Reason: %v`,
				statefulSetName,
				err,
			)
			log.Errorln(err)
		}
	}

	if !reflect.DeepEqual(updatedElastic.Spec.BackupSchedule, oldElastic.Spec.BackupSchedule) {
		c.ensureBackupScheduler(updatedElastic)
	}

	if !reflect.DeepEqual(oldElastic.Spec.Monitor, updatedElastic.Spec.Monitor) {
		if err := c.updateMonitor(oldElastic, updatedElastic); err != nil {
			c.eventRecorder.Eventf(
				updatedElastic,
				kapi.EventTypeWarning,
				eventer.EventReasonFailedToUpdateMonitor,
				"Failed to update monitoring system. Reason: %v",
				err,
			)
			log.Errorln(err)
			return nil
		}
		c.eventRecorder.Event(
			updatedElastic,
			kapi.EventTypeNormal,
			eventer.EventReasonSuccessfulMonitorUpdate,
			"Successfully updated monitoring system.",
		)
	}

	return nil
}
