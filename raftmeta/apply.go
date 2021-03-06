package raftmeta

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/angopher/chronus/raftmeta/internal"
	"github.com/angopher/chronus/x"
	"github.com/influxdata/influxdb/services/meta"
	"go.uber.org/zap"
)

func (s *RaftNode) applyCommitted(proposal *internal.Proposal, index uint64) error {
	if s.ApplyCallBack != nil {
		//only for test
		s.ApplyCallBack(proposal, index)
	}
	msgName, _ := internal.MessageTypeName[proposal.Type]
	s.Logger.Debug("applyCommitted ", zap.String("type", msgName))

	pctx := s.props.pctx(proposal.Key)
	if pctx == nil {
		pctx = &proposalCtx{}
	}
	pctx.index = index

	switch proposal.Type {
	case internal.CreateDatabase:
		var req CreateDatabaseReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Infof("apply create database %+v", req)
		db, err := s.MetaStore.CreateDatabase(req.Name)
		pctx.err = err
		if err == nil && pctx.retData != nil {
			x.AssertTrue(db != nil)
			*pctx.retData.(*meta.DatabaseInfo) = *db //TODO:db pointer是否有风险?
		}
		return err
	case internal.DropDatabase:
		var req DropDatabaseReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.DropDatabase(req.Name)
	case internal.DropRetentionPolicy:
		var req DropRetentionPolicyReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.DropRetentionPolicy(req.Database, req.Policy)
	case internal.CreateShardGroup:
		var req CreateShardGroupReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		sg, err := s.MetaStore.CreateShardGroup(req.Database, req.Policy, time.Unix(req.Timestamp, 0))
		if err == nil && pctx.retData != nil {
			if sg != nil {
				*pctx.retData.(*meta.ShardGroupInfo) = *sg
			} else {
				return errors.New("create shard group fail. have no data nodes available")
			}
		}
		return err
	case internal.CreateDataNode:
		var req CreateDataNodeReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		ni, err := s.MetaStore.CreateDataNode(req.HttpAddr, req.TcpAddr)
		if err == nil && pctx.retData != nil {
			x.AssertTrue(ni != nil)
			*pctx.retData.(*meta.NodeInfo) = *ni
		}
		return err
	case internal.DeleteDataNode:
		var req DeleteDataNodeReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.DeleteDataNode(req.Id)
	case internal.CreateRetentionPolicy:
		var req CreateRetentionPolicyReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)

		var duration *time.Duration
		if req.Rps.Duration > 0 {
			duration = &req.Rps.Duration
		}

		spec := meta.RetentionPolicySpec{
			Name:               req.Rps.Name,
			ReplicaN:           &req.Rps.ReplicaN,
			Duration:           duration,
			ShardGroupDuration: req.Rps.ShardGroupDuration,
		}
		rpi, err := s.MetaStore.CreateRetentionPolicy(req.Database, &spec, req.MakeDefault)
		if err == nil && pctx.retData != nil {
			x.AssertTrue(rpi != nil)
			*pctx.retData.(*meta.RetentionPolicyInfo) = *rpi
		}
		return err

	case internal.UpdateRetentionPolicy:
		var req UpdateRetentionPolicyReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)

		var duration *time.Duration
		if req.Rps.Duration > 0 {
			duration = &req.Rps.Duration
		}

		var sduration *time.Duration
		if req.Rps.ShardGroupDuration > 0 {
			sduration = &req.Rps.ShardGroupDuration
		}

		var rpName *string
		if req.Rps.Name != "" {
			rpName = &req.Rps.Name
		}

		up := meta.RetentionPolicyUpdate{
			Name:               rpName,
			ReplicaN:           &req.Rps.ReplicaN,
			Duration:           duration,
			ShardGroupDuration: sduration,
		}
		return s.MetaStore.UpdateRetentionPolicy(req.Database, req.Name, &up, req.MakeDefault)

	case internal.CreateDatabaseWithRetentionPolicy:
		var req CreateDatabaseWithRetentionPolicyReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)

		var duration *time.Duration
		if req.Rps.Duration > 0 {
			duration = &req.Rps.Duration
		}

		spec := meta.RetentionPolicySpec{
			Name:               req.Rps.Name,
			ReplicaN:           &req.Rps.ReplicaN,
			Duration:           duration,
			ShardGroupDuration: req.Rps.ShardGroupDuration,
		}
		db, err := s.MetaStore.CreateDatabaseWithRetentionPolicy(req.Name, &spec)
		if err == nil && pctx.retData != nil {
			x.AssertTrue(db != nil)
			*pctx.retData.(*meta.DatabaseInfo) = *db
		}
		return err

	case internal.CreateUser:
		var req CreateUserReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		user, err := s.MetaStore.CreateUser(req.Name, req.Password, req.Admin)
		if err == nil && pctx.retData != nil {
			x.AssertTrue(user != nil)
			*pctx.retData.(*meta.UserInfo) = *(user.(*meta.UserInfo))
		}
		return err

	case internal.DropUser:
		var req DropUserReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.DropUser(req.Name)

	case internal.UpdateUser:
		var req UpdateUserReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		// XXX password should be hashed before to keep consistent between nodes
		return s.MetaStore.UpdateUser(req.Name, req.Password)

	case internal.SetPrivilege:
		var req SetPrivilegeReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.SetPrivilege(req.UserName, req.Database, req.Privilege)

	case internal.SetAdminPrivilege:
		var req SetAdminPrivilegeReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.SetAdminPrivilege(req.UserName, req.Admin)

	case internal.Authenticate:
		var req AuthenticateReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		user, err := s.MetaStore.Authenticate(req.UserName, req.Password)
		if err == nil && pctx != nil && pctx.retData != nil {
			x.AssertTrue(user != nil)
			*pctx.retData.(*meta.UserInfo) = *(user.(*meta.UserInfo))
		}
		return err

	case internal.AddShardOwner:
		var req AddShardOwnerReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("add shard owner req %+v", req)
		return s.MetaStore.AddShardOwner(req.ShardID, req.NodeID)

	case internal.RemoveShardOwner:
		var req RemoveShardOwnerReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("remove shard owner req %+v", req)
		return s.MetaStore.RemoveShardOwner(req.ShardID, req.NodeID)

	case internal.DropShard:
		var req DropShardReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.DropShard(req.Id)

	case internal.TruncateShardGroups:
		var req TruncateShardGroupsReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.TruncateShardGroups(req.Time)

	case internal.PruneShardGroups:
		return s.MetaStore.PruneShardGroups()

	case internal.DeleteShardGroup:
		var req DeleteShardGroupReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.DeleteShardGroup(req.Database, req.Policy, req.Id)

	case internal.PrecreateShardGroups:
		var req PrecreateShardGroupsReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.PrecreateShardGroups(req.From, req.To)

	case internal.CreateContinuousQuery:
		var req CreateContinuousQueryReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.CreateContinuousQuery(req.Database, req.Name, req.Query)

	case internal.DropContinuousQuery:
		var req DropContinuousQueryReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.DropContinuousQuery(req.Database, req.Name)

	case internal.CreateSubscription:
		var req CreateSubscriptionReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		return s.MetaStore.CreateSubscription(req.Database, req.Rp, req.Name, req.Mode, req.Destinations)

	case internal.DropSubscription:
		var req DropSubscriptionReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debug("req %+v", req)
		return s.MetaStore.DropSubscription(req.Database, req.Rp, req.Name)

	case internal.AcquireLease:
		var req AcquireLeaseReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		if req.RequestTime == 0 {
			// XXX compatible with old data format
			req.RequestTime = time.Now().UnixNano() / 1e6
		}
		lease, err := s.leases.Acquire(req.Name, req.NodeId, req.RequestTime)
		if err == nil && pctx.retData != nil {
			x.AssertTrue(lease != nil)
			*pctx.retData.(*meta.Lease) = *lease
		}
		return err

	case internal.SnapShot:
		md, err := s.MetaStore.MarshalBinary()
		x.Check(err)
		var sndata internal.SnapshotData
		sndata.Data = md
		sndata.PeersAddr = s.Transport.ClonePeers()

		data, err := json.Marshal(&sndata)
		x.Check(err)

		conf := s.ConfState()
		return s.Storage.CreateSnapshot(index, conf, data)

	case internal.CreateChecksumMsg:
		//TODO:optimize, reduce block time
		start := time.Now()
		mcd := s.MetaStore.Data()

		//消除DeleteAt和TruncatedAt对checksum的影响
		for i := range mcd.Databases {
			db := &mcd.Databases[i]
			for j := range db.RetentionPolicies {
				rp := &db.RetentionPolicies[j]
				for k := range rp.ShardGroups {
					sg := &rp.ShardGroups[k]
					sg.DeletedAt = time.Unix(0, 0)
					sg.TruncatedAt = time.Unix(0, 0)
				}
			}
		}
		data, err := (&mcd).MarshalBinary()
		x.Check(err)
		s.lastChecksum.index = index
		s.lastChecksum.checksum = x.Md5(data)
		s.lastChecksum.needVerify = true

		s.Logger.Debug(
			fmt.Sprintf("create checksum costs:%s detail:%s",
				time.Now().Sub(start),
				fmt.Sprintf("index:%d, checksum:%s, data:%+v",
					index, s.lastChecksum.checksum, mcd,
				),
			),
		)
	case internal.VerifyChecksumMsg:
		start := time.Now()
		var req internal.VerifyChecksum
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		if req.NodeID == s.ID {
			s.Logger.Debug("ignore checksum. self trigger this verify")
			s.lastChecksum.needVerify = false
			return nil
		}

		if s.lastChecksum.index == 0 {
			//have no checksum only when restart
			s.Logger.Warn("ignore checksum. have no checksum", zap.Uint64("index", req.Index))
			return nil
		}

		if s.lastChecksum.index != req.Index {
			s.Logger.Warn("ignore checksum", zap.Uint64("last index:", s.lastChecksum.index), zap.Uint64("index", req.Index))
			return nil
		}

		s.Logger.Info("checksum", zap.Uint64("index", req.Index), zap.String("checksum", s.lastChecksum.checksum))
		x.AssertTruef(s.lastChecksum.checksum == req.Checksum, "verify checksum fail, local %s != %s, data=%+v",
			s.lastChecksum.checksum, req.Checksum,
			s.MetaStore.Data(),
		)
		s.Logger.Info(fmt.Sprintf("verify checksum success. costs %s", time.Now().Sub(start)))
	case internal.FreezeDataNode:
		var req FreezeDataNodeReq
		err := json.Unmarshal(proposal.Data, &req)
		x.Check(err)
		s.SugaredLogger.Debugf("req %+v", req)
		if req.Freeze {
			return s.MetaStore.FreezeDataNode(req.Id)
		} else {
			return s.MetaStore.UnfreezeDataNode(req.Id)
		}
	default:
		return fmt.Errorf("Unknown msg type:%d", proposal.Type)
	}

	return nil
}
