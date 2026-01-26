import * as React from 'react';
import { useApi, identityApiRef } from '@backstage/core-plugin-api';
import { Avatar } from '@backstage/core-components';
import useAsync from 'react-use/lib/useAsync';
import { makeStyles } from '@material-ui/core/styles';

const useStyles = makeStyles({
    avatarContainer: {
        position: 'fixed',
        top: 16,
        right: 16,
        zIndex: 1000,
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        cursor: 'pointer',
    },
    displayName: {
        color: '#fff',
        fontWeight: 'bold',
        textShadow: '0 1px 2px rgba(0,0,0,0.5)',
    }
});

export const UserHeader = () => {
    const classes = useStyles();
    const identity = useApi(identityApiRef);

    const { value: profile } = useAsync(async () => {
        return await identity.getProfileInfo();
    }, []);

    if (!profile) return null;

    return (
        <div className={classes.avatarContainer}>
            {/* Optional: Show name too? */}
            {/* <span className={classes.displayName}>{profile.displayName}</span> */}
            <Avatar
                picture={profile.picture}
                displayName={profile.displayName || profile.email}
                customStyles={{ width: 40, height: 40, border: '2px solid #fff' }}
            />
        </div>
    );
};
