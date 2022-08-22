package handler

import (
	"encoding/json"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/internal/core/application"
	neutrinodtypes "github.com/vulpemventures/neutrino-elements/pkg/neutrinod-types"
	"net/http"
)

func (d *descriptorWalletNotifierHandler) HandleSubscriptionRequestHttp(
	w http.ResponseWriter,
	req *http.Request,
) {
	var subscriptionReq neutrinodtypes.SubscriptionRequestHttp

	err := json.NewDecoder(req.Body).Decode(&subscriptionReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	subsID := uuid.New()
	d.registerSubs <- &HttpSubscriber{
		ID:          SubscriberID(subsID),
		EndpointUrl: subscriptionReq.EndpointUrl,
	}

	switch subscriptionReq.ActionType {
	case neutrinodtypes.Register:
		if err := d.notificationSvc.Subscribe(application.Subscriber{
			ID:               application.SubscriberID(subsID),
			BlockHeight:      subscriptionReq.StartBlockHeight,
			Events:           subscriptionReq.EventTypes,
			WalletDescriptor: subscriptionReq.DescriptorWallet,
		}); err != nil {
			log.Errorf("unsucesfull registration: %v, subscriber: %v", err, subsID)

			resp := neutrinodtypes.MessageErrorResponse{
				ErrorMessage: "un-successful registration",
			}
			sendResponseToSubscriberHttp(w, resp)
		}
		log.Infof("sucesfull registration, subscriber: %v", subsID)

		resp := neutrinodtypes.GeneralMessageResponse{
			Message: "successful registration",
		}
		sendResponseToSubscriberHttp(w, resp)
	case neutrinodtypes.Unregister:
		if err := d.notificationSvc.UnSubscribe(application.Subscriber{
			ID: application.SubscriberID(subsID),
		}); err != nil {
			log.Errorf("unsucesfull un-registration: %v, subscriber: %v", err, subsID)

			resp := neutrinodtypes.MessageErrorResponse{
				ErrorMessage: "un-successful un-registration",
			}
			sendResponseToSubscriberHttp(w, resp)
		}

		log.Infof("sucesfull un-registration: %v, subscriber: %v", err, subsID)

		resp := neutrinodtypes.GeneralMessageResponse{
			Message: "successful un-registration",
		}
		sendResponseToSubscriberHttp(w, resp)
	default:
		log.Errorf("unknown action type: %v\n", subscriptionReq.ActionType)

		resp := neutrinodtypes.MessageErrorResponse{
			ErrorMessage: "bad request",
		}
		sendResponseToSubscriberHttp(w, resp)
	}

	log.Debugf("new ws subscriber connected: %v", subsID)
}

func sendResponseToSubscriberHttp[
V neutrinodtypes.MessageErrorResponse |
neutrinodtypes.OnChainEventResponse |
neutrinodtypes.GeneralMessageResponse](
	w http.ResponseWriter,
	resp V,
) {
	respJson, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("handler error -> json marshal: %v", err)
		return
	}

	_, err = w.Write(respJson)
	if err != nil {
		log.Errorf("handleError -> failed writing to response: %v", err.Error())
		return
	}
}
