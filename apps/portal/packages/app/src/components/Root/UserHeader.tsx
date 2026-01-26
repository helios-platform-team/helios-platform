import { useApi, identityApiRef } from '@backstage/core-plugin-api';
import { Avatar } from '@backstage/core-components';
import useAsync from 'react-use/lib/useAsync';
import { makeStyles } from '@material-ui/core/styles';

const useStyles = makeStyles({
    avatarContainer: {
        position: 'fixed',
        top: 12,
        right: 64, // Moved left to avoid obscuring the 3-dots menu
        zIndex: 1000,
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '4px 12px 4px 4px',
        borderRadius: 30,
        backgroundColor: 'rgba(0, 0, 0, 0.15)', // Glassmorphism-ish background
        backdropFilter: 'blur(4px)',
        border: '1px solid rgba(255, 255, 255, 0.1)',
        cursor: 'pointer',
        transition: 'all 0.2s ease-in-out',
        '&:hover': {
            backgroundColor: 'rgba(0, 0, 0, 0.25)',
            transform: 'translateY(1px)',
        },
    },
    displayName: {
        color: '#fff',
        fontWeight: 600,
        fontSize: '0.875rem',
        textShadow: '0 1px 2px rgba(0,0,0,0.5)',
        paddingRight: 4,
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
            <Avatar
                picture={profile.picture}
                displayName={profile.displayName || profile.email}
                customStyles={{ width: 32, height: 32, border: 'none', boxShadow: '0 2px 4px rgba(0,0,0,0.2)' }}
            />
            <span className={classes.displayName}>
                {profile.displayName || profile.email || 'User'}
            </span>
        </div>
    );
};
