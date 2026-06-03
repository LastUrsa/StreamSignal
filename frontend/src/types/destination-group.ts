export const DEFAULT_DESTINATION_GROUP_ID = 'all-destinations';
export const DEFAULT_DESTINATION_GROUP_NAME = 'All Destinations';

export interface DestinationGroupInput {
    id: string;
    name: string;
    description: string;
    createdAt?: string;
    updatedAt?: string;
}

export interface DestinationGroupFormState {
    id: string;
    name: string;
    description: string;
}

export function createEmptyDestinationGroupForm(): DestinationGroupFormState {
    return {
        id: '',
        name: '',
        description: '',
    };
}

export function toDestinationGroupFormState(group: DestinationGroupInput): DestinationGroupFormState {
    return {
        id: group.id,
        name: group.name,
        description: group.description,
    };
}
