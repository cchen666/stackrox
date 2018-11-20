import { all, take, takeLatest, call, fork, put, select, cancel } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { networkPath } from 'routePaths';
import * as service from 'services/NetworkService';
import { fetchClusters } from 'services/ClustersService';
import { actions, types } from 'reducers/network';
import { actions as clusterActions } from 'reducers/clusters';
import { actions as notificationActions } from 'reducers/notifications';
import { selectors } from 'reducers';
import { takeEveryLocation } from 'utils/sagaEffects';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { types as deploymentTypes } from 'reducers/deployments';
import { types as locationActionTypes } from 'reducers/routes';
import { getDeployment } from './deploymentSagas';

function* getNetworkFlowGraph(filters, clusterId) {
    yield put(actions.fetchNetworkFlowGraph.request());
    try {
        const flowResult = yield call(service.fetchNetworkFlowGraph, filters, clusterId);
        yield put(actions.fetchNetworkFlowGraph.success(flowResult.response));
        yield put(actions.setNetworkFlowMapping(flowResult.response));
        yield put(actions.updateNetworkGraphTimestamp(new Date()));
    } catch (error) {
        yield put(actions.fetchNetworkFlowGraph.failure(error));
    }
}

function* getNetworkGraph(filters, clusterId) {
    yield put(actions.fetchNetworkPolicyGraph.request());
    try {
        const policyResult = yield call(service.fetchNetworkPolicyGraph, filters, clusterId);
        yield put(actions.fetchNetworkPolicyGraph.success(policyResult.response));
        yield put(actions.updateNetworkGraphTimestamp(new Date()));
        yield fork(getNetworkFlowGraph, filters, clusterId);
    } catch (error) {
        yield put(actions.fetchNetworkPolicyGraph.failure(error));
    }
}

function* getSelectedDeployment({ params }) {
    yield call(getDeployment, params);
}

export function* getNetworkPolicies({ params }) {
    try {
        const result = yield call(service.fetchNetworkPolicies, params);
        yield put(actions.fetchNetworkPolicies.success(result.response, { params }));
    } catch (error) {
        yield put(actions.fetchNetworkPolicies.failure(error));
    }
}

export function* pollNodeUpdates() {
    while (true) {
        try {
            const result = yield call(service.fetchNodeUpdates);
            yield put(actions.fetchNodeUpdates.success(result.response));
        } catch (error) {
            yield put(actions.fetchNodeUpdates.failure(error));
        }
        yield call(delay, 5000); // poll every 5 sec
    }
}

function* sendYAMLNotification({ notifierId }) {
    try {
        const clusterId = yield select(selectors.getSelectedNetworkClusterId);
        const { content } = yield select(selectors.getYamlFile);
        yield call(service.sendYAMLNotification, clusterId, notifierId, content);
        yield put(notificationActions.addNotification('Successfully sent notification.'));
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
    }
}

function* watchLocation() {
    let pollTask = null;
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (
            location &&
            location.pathname &&
            location.pathname.startsWith(networkPath) &&
            !pollTask
        ) {
            // start only if it's not already in progress
            pollTask = yield fork(pollNodeUpdates);
        } else if (pollTask) {
            yield cancel(pollTask);
            pollTask = null;
            yield put(actions.setSelectedNodeId(null));
            yield put(actions.setNetworkGraphState(null));
        }
    }
}

function* filterNetworkPageBySearch() {
    const clusterId = yield select(selectors.getSelectedNetworkClusterId);
    const searchOptions = yield select(selectors.getNetworkSearchOptions);
    const yamlFile = yield select(selectors.getYamlFile);
    if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
        return;
    }
    const filters = {
        query: searchOptionsToQuery(searchOptions)
    };
    if (yamlFile) {
        filters.simulationYaml = yamlFile.content;
    }
    if (clusterId) {
        yield fork(getNetworkGraph, filters, clusterId);
    }
}

function* loadNetworkPage() {
    try {
        const result = yield call(fetchClusters);
        yield put(clusterActions.fetchClusters.success(result.response));
        yield put(actions.selectDefaultNetworkClusterId(result.response));
        yield fork(filterNetworkPageBySearch);
    } catch (error) {
        yield put(clusterActions.fetchClusters.failure(error));
    }
}

function* watchNetworkSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, filterNetworkPageBySearch);
}

function* watchFetchDeploymentRequest() {
    yield takeLatest(deploymentTypes.FETCH_DEPLOYMENT.REQUEST, getSelectedDeployment);
}

function* watchNetworkPoliciesRequest() {
    yield takeLatest(types.FETCH_NETWORK_POLICIES.REQUEST, getNetworkPolicies);
}

function* watchSelectNetworkCluster() {
    yield takeLatest(types.SELECT_NETWORK_CLUSTER_ID, filterNetworkPageBySearch);
}

function* watchSendYAMLNotification() {
    yield takeLatest(types.SEND_YAML_NOTIFICATION, sendYAMLNotification);
}

function* watchSetYamlFile() {
    yield takeLatest(types.SET_YAML_FILE, filterNetworkPageBySearch);
}

function* watchNetworkNodesUpdate() {
    yield takeLatest(types.NETWORK_NODES_UPDATE, filterNetworkPageBySearch);
}

export default function* network() {
    yield all([
        takeEveryLocation(networkPath, loadNetworkPage),
        fork(watchNetworkSearchOptions),
        fork(watchNetworkPoliciesRequest),
        fork(watchFetchDeploymentRequest),
        fork(watchSelectNetworkCluster),
        fork(watchNetworkNodesUpdate),
        fork(watchSetYamlFile),
        fork(watchSendYAMLNotification),
        fork(watchLocation)
    ]);
}
