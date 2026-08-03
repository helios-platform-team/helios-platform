import {
  Button,
  Grid,
  Avatar,
  Typography,
  Box,
  makeStyles,
} from '@material-ui/core';
import { useApi, identityApiRef } from '@backstage/core-plugin-api';
import useAsync from 'react-use/lib/useAsync';
import {
  SettingsLayout,
  UserSettingsAppearanceCard,
  UserSettingsIdentityCard,
  UserSettingsAuthProviders,
} from '@backstage/plugin-user-settings';
import ExitToAppIcon from '@material-ui/icons/ExitToApp';
import { InfoCard } from '@backstage/core-components';

const useStyles = makeStyles(theme => ({
  profileContainer: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(3),
  },
  avatar: {
    width: theme.spacing(10),
    height: theme.spacing(10),
    fontSize: '2rem',
  },
  profileInfo: {
    flexGrow: 1,
  },
}));

const CustomProfileCard = () => {
  const classes = useStyles();
  const identity = useApi(identityApiRef);
  const { value: profile, loading } = useAsync(
    () => identity.getProfileInfo(),
    [],
  );

  if (loading) {
    return <InfoCard title="Profile">Loading...</InfoCard>;
  }

  const displayName = profile?.displayName || 'User';
  const email = profile?.email || '';
  const picture = profile?.picture;

  return (
    <InfoCard title="Profile">
      <div className={classes.profileContainer}>
        <Avatar src={picture} className={classes.avatar} alt={displayName}>
          {displayName.charAt(0).toUpperCase()}
        </Avatar>
        <Box className={classes.profileInfo}>
          <Typography variant="h5">{displayName}</Typography>
          {email && (
            <Typography variant="body1" color="textSecondary">
              {email}
            </Typography>
          )}
        </Box>
        <Button
          variant="outlined"
          color="secondary"
          startIcon={<ExitToAppIcon />}
          onClick={() => identity.signOut()}
        >
          Sign Out
        </Button>
      </div>
    </InfoCard>
  );
};

export const CustomSettingsPage = () => {
  return (
    <SettingsLayout>
      <SettingsLayout.Route path="general" title="General">
        <Grid container direction="column" spacing={3}>
          <Grid item>
            <CustomProfileCard />
          </Grid>
          <Grid item>
            <UserSettingsAppearanceCard />
          </Grid>
          <Grid item>
            <UserSettingsIdentityCard />
          </Grid>
        </Grid>
      </SettingsLayout.Route>
      <SettingsLayout.Route path="advanced" title="Advanced">
        <UserSettingsAuthProviders />
      </SettingsLayout.Route>
    </SettingsLayout>
  );
};
