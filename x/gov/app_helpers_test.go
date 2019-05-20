package gov

import (
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

// Test if the helper class AppCreateProposalMsg is byte compatible with CreateProposalMsg
func TestAppCreateProposalMsg(t *testing.T) {
	electorateOpts := &ProposalOptions{
		Option: &ProposalOptions_Electorate{
			Electorate: &UpdateElectorateMsg{
				Metadata:     &weave.Metadata{Schema: 1},
				ElectorateID: weavetest.SequenceID(1),
				DiffElectors: []Elector{{
					Address: weavetest.NewCondition().Address(),
					Weight:  22,
				}},
			},
		},
	}
	electorateBytes, err := electorateOpts.Marshal()
	assert.Nil(t, err)

	now := weave.AsUnixTime(time.Now())
	proposalMsg := CreateProposalMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Base: &CreateProposalMsgBase{
			Title:          "my proposal",
			Description:    "my description",
			StartTime:      now.Add(time.Hour),
			ElectionRuleID: weavetest.SequenceID(1),
			Author:         hBobby,
		},
		RawOption: electorateBytes,
	}
	appProposalMsg := AppCreateProposalMsg{
		Metadata: proposalMsg.Metadata,
		Base:     proposalMsg.Base,
		Options:  electorateOpts,
	}

	propBytes, err := proposalMsg.Marshal()
	assert.Nil(t, err)
	appBytes, err := appProposalMsg.Marshal()
	assert.Nil(t, err)

	assert.Equal(t, propBytes, appBytes)

	var loadMsg CreateProposalMsg
	err = loadMsg.Unmarshal(appBytes)
	assert.Nil(t, err)
	assert.Equal(t, proposalMsg, loadMsg)
}

// Test if the helper class AppProposal is byte compatible with Proposal
func TestAppProposal(t *testing.T) {
	alice := weavetest.NewCondition().Address()
	proposal := proposalFixture(alice)

	appProposal := AppProposal{
		Metadata: proposal.Metadata,
		Common:   proposal.Common,
		Options: &ProposalOptions{
			Option: &ProposalOptions_Text{
				Text: &TextResolutionMsg{
					Metadata:   &weave.Metadata{Schema: 1},
					Resolution: "Lower tx fees for all!",
				},
			},
		},
	}

	propBytes, err := proposal.Marshal()
	assert.Nil(t, err)
	appBytes, err := appProposal.Marshal()
	assert.Nil(t, err)

	assert.Equal(t, propBytes, appBytes)

	var loadProp Proposal
	err = loadProp.Unmarshal(appBytes)
	assert.Nil(t, err)
	assert.Equal(t, proposal, loadProp)
}
