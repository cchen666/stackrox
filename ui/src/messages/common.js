const common = Object.freeze({
    severityLabels: {
        CRITICAL_SEVERITY: 'Critical',
        HIGH_SEVERITY: 'High',
        MEDIUM_SEVERITY: 'Medium',
        LOW_SEVERITY: 'Low'
    },
    clusterTypeLabels: {
        KUBERNETES_CLUSTER: 'Kubernetes Clusters',
        SWARM_CLUSTER: 'Swarm Clusters',
        OPENSHIFT_CLUSTER: 'OpenShift Clusters'
    },
    clusterVersionLabels: {
        KUBERNETES_CLUSTER: 'K8s Version',
        SWARM_CLUSTER: 'Swarm Version',
        OPENSHIFT_CLUSTER: 'OpenShift Version'
    },
    lifecycleStageLabels: {
        BUILD: 'Build',
        DEPLOY: 'Deploy',
        RUNTIME: 'Runtime'
    },
    enforcementActionLabels: {
        UNSET_ENFORCEMENT: 'None',
        FAIL_BUILD_ENFORCEMENT: 'Fail builds during continuous integration',
        SCALE_TO_ZERO_ENFORCEMENT: 'Scale to Zero Replicas',
        KILL_POD_ENFORCEMENT: 'Kill Pod',
        UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: 'Add an Unsatisfiable Node Constraint'
    },
    accessControl: {
        NO_ACCESS: 'No Access',
        READ_ACCESS: 'Read Access',
        READ_WRITE_ACCESS: 'Read and Write Access'
    },
    resourceLabels: {
        CLUSTER: 'cluster',
        NAMESPACE: 'namespace',
        NODE: 'node',
        DEPLOYMENT: 'deployment',
        SECRET: 'secret',
        CONTROL: 'control',
        IMAGE: 'image',
        POLICY: 'policy'
    },
    rbacConfigLabels: {
        SUBJECT: 'users and groups',
        SERVICE_ACCOUNT: 'serviceaccount',
        ROLE: 'role'
    },
    stackroxSupport: {
        phoneNumber: {
            withSpaces: '1 (650) 489-6769',
            withDashes: '1-650-489-6769'
        },
        email: 'support@stackrox.com'
    },
    portExposureLabels: {
        EXTERNAL: 'LoadBalancer',
        NODE: 'NodePort',
        HOST: 'HostPort',
        INTERNAL: 'ClusterIP'
    },
    rbacPermissionLabels: {
        NONE: 'No Access',
        DEFAULT: 'Default Access',
        ELEVATED_IN_NAMESPACE: 'Elevated Access in Namespace',
        ELEVATED_CLUSTER_WIDE: 'Elevated Access Cluster Wide',
        CLUSTER_ADMIN: 'Cluster Admin Access'
    }
});

module.exports = common;
