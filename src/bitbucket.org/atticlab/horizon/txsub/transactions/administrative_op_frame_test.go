package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/admin"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/test"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestAdministrativeOpFrame(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	historyQ := &history.Q{
		tt.HorizonRepo(),
	}
	coreQ := &core.Q{
		tt.CoreRepo(),
	}
	config := test.NewTestConfig()

	root := test.BankMasterSeed()

	Convey("Invalid OpData:", t, func() {
		adminOp := build.AdministrativeOp(build.OpLongData{"random_data"})
		tx := build.Transaction(adminOp, build.Sequence{1}, build.SourceAccount{root.Address()})
		txEBuilder := tx.Sign(root.Seed())
		opFrame := NewOperationFrame(&txEBuilder.E.Tx.Operations[0], txEBuilder.E)
		isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustAdminResult().Code, ShouldEqual, xdr.AdministrativeResultCodeAdministrativeMalformed)
	})
	Convey("Valid OpData", t, func() {
		adminOp := build.AdministrativeOp(build.OpLongData{"{}"})
		tx := build.Transaction(adminOp, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := tx.Sign(root.Seed()).E
		opFrame := NewOperationFrame(&txE.Tx.Operations[0], txE)
		Convey("Unknown admin action", func() {
			adminActionProviderM := admin.AdminActionProviderMock{}
			errorData := "unknown admin action"
			adminActionProviderM.On("CreateNewParser", mock.Anything).Return(nil, errors.New(errorData))
			adminOpFrame := GetAdminOpFrame(&opFrame)
			adminOpFrame.adminActionProvider = &adminActionProviderM
			isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeFalse)
			So(opFrame.GetResult().Result.MustTr().MustAdminResult().Code, ShouldEqual, xdr.AdministrativeResultCodeAdministrativeMalformed)
			So(opFrame.GetResult().Info.GetError(), ShouldEqual, errorData)
		})
		Convey("Invalid field", func() {
			adminActionMock := admin.AdminActionMock{}
			invalidField := admin.InvalidField("invalid_field_name", errors.New("error"))
			adminActionMock.On("GetError").Return(invalidField)
			adminActionProviderM := admin.AdminActionProviderMock{}
			adminActionProviderM.On("CreateNewParser", mock.Anything).Return(&adminActionMock, nil)
			adminOpFrame := GetAdminOpFrame(&opFrame)
			adminOpFrame.adminActionProvider = &adminActionProviderM
			isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeFalse)
			So(opFrame.GetResult().Result.MustTr().MustAdminResult().Code, ShouldEqual, xdr.AdministrativeResultCodeAdministrativeMalformed)
			So(opFrame.GetResult().Info.GetError(), ShouldEqual, invalidField.Reason.Error())
			So(opFrame.GetResult().Info.GetInvalidField(), ShouldEqual, invalidField.FieldName)
		})
		Convey("Server error", func() {
			adminActionMock := admin.AdminActionMock{}
			adminActionMock.On("GetError").Return(&problem.ServerError)
			adminActionProviderM := admin.AdminActionProviderMock{}
			adminActionProviderM.On("CreateNewParser", mock.Anything).Return(&adminActionMock, nil)
			adminOpFrame := GetAdminOpFrame(&opFrame)
			adminOpFrame.adminActionProvider = &adminActionProviderM
			isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
			assert.Equal(t, &problem.ServerError, err)
			So(isValid, ShouldBeFalse)
		})
		Convey("Problem", func() {
			adminActionMock := admin.AdminActionMock{}
			adminActionMock.On("GetError").Return(&problem.BadRequest)
			adminActionProviderM := admin.AdminActionProviderMock{}
			adminActionProviderM.On("CreateNewParser", mock.Anything).Return(&adminActionMock, nil)
			adminOpFrame := GetAdminOpFrame(&opFrame)
			adminOpFrame.adminActionProvider = &adminActionProviderM
			isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeFalse)
			So(opFrame.GetResult().Result.MustTr().MustAdminResult().Code, ShouldEqual, xdr.AdministrativeResultCodeAdministrativeMalformed)
		})
		Convey("Error", func() {
			adminActionMock := admin.AdminActionMock{}
			errData := "error"
			adminActionMock.On("GetError").Return(errors.New(errData))
			adminActionProviderM := admin.AdminActionProviderMock{}
			adminActionProviderM.On("CreateNewParser", mock.Anything).Return(&adminActionMock, nil)
			adminOpFrame := GetAdminOpFrame(&opFrame)
			adminOpFrame.adminActionProvider = &adminActionProviderM
			isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
			So(err.Error(), ShouldEqual, errData)
			So(isValid, ShouldBeFalse)
		})
		Convey("Success", func() {
			adminActionMock := admin.AdminActionMock{}
			adminActionMock.On("GetError").Return(nil)
			adminActionProviderM := admin.AdminActionProviderMock{}
			adminActionProviderM.On("CreateNewParser", mock.Anything).Return(&adminActionMock, nil)
			adminOpFrame := GetAdminOpFrame(&opFrame)
			adminOpFrame.adminActionProvider = &adminActionProviderM
			isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeTrue)
			So(opFrame.GetResult().Result.MustTr().MustAdminResult().Code, ShouldEqual, xdr.AdministrativeResultCodeAdministrativeSuccess)
		})
	})
}

func GetAdminOpFrame(opFrame *OperationFrame) *AdministrativeOpFrame {
	innerOp, err := opFrame.GetInnerOp()
	if err != nil || innerOp == nil {
		log.Panic("Failed to create innerOp")
	}

	return innerOp.(*AdministrativeOpFrame)
}
