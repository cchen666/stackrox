import React, { ReactElement } from 'react';
import { Flex, TextInput, FormGroup, Radio, TextArea } from '@patternfly/react-core';
import { Field } from 'formik';

import PolicyCategoriesSelectField from './PolicyCategoriesSelectField';

type PolicyMetadataFormSectionProps = {
    handleChange: (e) => void;
};

function PolicyMetadataFormSection({ handleChange }: PolicyMetadataFormSectionProps): ReactElement {
    function onChange(_value, event) {
        handleChange(event);
    }
    return (
        <>
            <Field name="name">
                {({ field }) => (
                    <FormGroup
                        helperText="Provide a descriptive and unique policy name"
                        fieldId="policy-name"
                        label="Name"
                        className="pf-u-pt-md"
                        isRequired
                    >
                        <TextInput
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={onChange}
                            isRequired
                        />
                    </FormGroup>
                )}
            </Field>
            <FormGroup
                helperText="Select a severity level for this policy"
                fieldId="policy-severity"
                label="Severity"
                className="pf-u-pt-md"
                isRequired
            >
                <Flex direction={{ default: 'row' }}>
                    <Field name="severity" type="radio" value="LOW_SEVERITY">
                        {({ field }) => (
                            <Radio
                                name={field.name}
                                value={field.value}
                                onChange={onChange}
                                label="Low"
                                id="policy-severity-radio-low"
                                isChecked={field.checked}
                            />
                        )}
                    </Field>
                    <Field name="severity" type="radio" value="MEDIUM_SEVERITY">
                        {({ field }) => (
                            <Radio
                                name={field.name}
                                value={field.value}
                                onChange={onChange}
                                label="Medium"
                                id="policy-severity-radio-medium"
                                isChecked={field.checked}
                            />
                        )}
                    </Field>
                    <Field name="severity" type="radio" value="HIGH_SEVERITY">
                        {({ field }) => (
                            <Radio
                                name={field.name}
                                value={field.value}
                                onChange={onChange}
                                label="High"
                                id="policy-severity-radio-high"
                                isChecked={field.checked}
                            />
                        )}
                    </Field>
                    <Field name="severity" type="radio" value="CRITICAL_SEVERITY">
                        {({ field }) => (
                            <Radio
                                name={field.name}
                                value={field.value}
                                onChange={onChange}
                                label="Critical"
                                id="policy-severity-radio-critical"
                                isChecked={field.checked}
                            />
                        )}
                    </Field>
                </Flex>
            </FormGroup>
            <PolicyCategoriesSelectField />
            <Field name="description">
                {({ field }) => (
                    <FormGroup
                        helperText="Enter details about the policy"
                        fieldId="policy-description"
                        label="Description"
                        className="pf-u-pt-md"
                        isRequired
                    >
                        <TextArea
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={onChange}
                            isRequired
                        />
                    </FormGroup>
                )}
            </Field>
            <Field name="rationale">
                {({ field }) => (
                    <FormGroup
                        helperText="Enter an explanation about why this policy exists"
                        fieldId="policy-rationale"
                        label="Rationale"
                        className="pf-u-pt-md"
                        isRequired
                    >
                        <TextArea
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={onChange}
                            isRequired
                        />
                    </FormGroup>
                )}
            </Field>
            <Field name="remediation">
                {({ field }) => (
                    <FormGroup
                        helperText="Enter steps to resolve the violations of this policy"
                        fieldId="policy-guidance"
                        label="Guidance"
                        className="pf-u-pt-md"
                        isRequired
                    >
                        <TextArea
                            id={field.name}
                            name={field.name}
                            value={field.value}
                            onChange={onChange}
                            isRequired
                        />
                    </FormGroup>
                )}
            </Field>
        </>
    );
}

export default PolicyMetadataFormSection;
