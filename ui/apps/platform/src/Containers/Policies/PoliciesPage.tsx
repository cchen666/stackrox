import React from 'react';
import { useLocation, useParams } from 'react-router-dom';

import usePermissions from 'hooks/usePermissions';
import useURLSearch from 'hooks/useURLSearch';

import NotFoundMessage from 'Components/NotFoundMessage';
import { parsePoliciesSearchString } from './policies.utils';
import PoliciesTablePage from './Table/PoliciesTablePage';
import PolicyPage from './PolicyPage';

function PoliciesPage() {
    /*
     * Examples of urls for PolicyPage:
     * /main/policies/:policyId
     * /main/policies/:policyId?action=edit
     * /main/policies?action=create
     *
     * Examples of urls for PolicyTablePage:
     * /main/policies
     * /main/policies?s[Lifecycle Stage]=BUILD
     * /main/policies?s[Lifecycle Stage]=BUILD&s[Lifecycle State]=DEPLOY
     * /main/policies?s[Lifecycle State]=RUNTIME&s[Severity]=CRITICAL_SEVERITY
     */
    const location = useLocation();
    const { search } = location;
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { pageAction } = parsePoliciesSearchString(search);
    const { policyId } = useParams();

    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasReadAccessForPolicy = hasReadAccess('Policy');
    const hasWriteAccessForPolicy = hasReadWriteAccess('Policy');

    if (!hasReadAccessForPolicy) {
        return <NotFoundMessage title="404: We couldn't find that page" />;
    }

    if (pageAction || policyId) {
        return (
            <PolicyPage
                hasWriteAccessForPolicy={hasWriteAccessForPolicy}
                pageAction={pageAction}
                policyId={policyId}
            />
        );
    }

    return (
        <PoliciesTablePage
            hasWriteAccessForPolicy={hasWriteAccessForPolicy}
            handleChangeSearchFilter={setSearchFilter}
            searchFilter={searchFilter}
        />
    );
}

export default PoliciesPage;
