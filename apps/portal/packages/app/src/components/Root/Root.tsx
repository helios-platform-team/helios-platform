import { PropsWithChildren } from 'react';
import { makeStyles } from '@material-ui/core';
import { useApi, identityApiRef } from '@backstage/core-plugin-api';
import useAsync from 'react-use/lib/useAsync';
import { Avatar } from '@backstage/core-components';
import HomeIcon from '@material-ui/icons/Home';
import ExtensionIcon from '@material-ui/icons/Extension';
import LibraryBooks from '@material-ui/icons/LibraryBooks';
import CreateComponentIcon from '@material-ui/icons/AddCircleOutline';
import LogoFull from './LogoFull';
import LogoIcon from './LogoIcon';
import {
  Settings as SidebarSettings,
  UserSettingsSignInAvatar,
} from '@backstage/plugin-user-settings';
import { SidebarSearchModal } from '@backstage/plugin-search';
import {
  Sidebar,
  sidebarConfig,
  SidebarDivider,
  SidebarGroup,
  SidebarItem,
  SidebarPage,
  SidebarScrollWrapper,
  SidebarSpace,
  useSidebarOpenState,
  Link,
} from '@backstage/core-components';
import MenuIcon from '@material-ui/icons/Menu';
import SearchIcon from '@material-ui/icons/Search';
import { MyGroupsSidebarItem } from '@backstage/plugin-org';
import GroupIcon from '@material-ui/icons/People';
import { NotificationsSidebarItem } from '@backstage/plugin-notifications';

const useSidebarLogoStyles = makeStyles({
  root: {
    width: sidebarConfig.drawerWidthClosed,
    height: 3 * sidebarConfig.logoHeight,
    display: 'flex',
    flexFlow: 'row nowrap',
    alignItems: 'center',
    marginBottom: -14,
  },
  link: {
    width: sidebarConfig.drawerWidthClosed,
    marginLeft: 24,
  },
});

const SidebarLogo = () => {
  const classes = useSidebarLogoStyles();
  const { isOpen } = useSidebarOpenState();

  return (
    <div className={classes.root}>
      <Link to="/" underline="none" className={classes.link} aria-label="Home">
        {isOpen ? <LogoFull /> : <LogoIcon />}
      </Link>
    </div>
  );
};

export const Root = ({ children }: PropsWithChildren<{}>) => {
  const identity = useApi(identityApiRef);
  const { value: profile } = useAsync(async () => {
    return await identity.getProfileInfo();
  }, []);

  return (
    <SidebarPage>
      <Sidebar>
        <SidebarLogo />
        <SidebarGroup label="Search" icon={<SearchIcon />} to="/search">
          <SidebarSearchModal />
        </SidebarGroup>
        <SidebarDivider />
        <SidebarGroup label="Menu" icon={<MenuIcon />}>
          {/* Global nav, not org-specific */}
          <SidebarItem icon={HomeIcon} to="catalog" text="Home" />
          <MyGroupsSidebarItem
            singularTitle="My Group"
            pluralTitle="My Groups"
            icon={GroupIcon}
          />
          <SidebarItem icon={ExtensionIcon} to="api-docs" text="APIs" />
          <SidebarItem icon={LibraryBooks} to="docs" text="Docs" />
          <SidebarItem
            icon={CreateComponentIcon}
            to="create"
            text="Create..."
          />
          {/* End global nav */}
          <SidebarDivider />
          <SidebarScrollWrapper>
            {/* Items in this group will be scrollable if they run out of space */}
          </SidebarScrollWrapper>
        </SidebarGroup>
        <SidebarSpace />
        <SidebarDivider />
        <NotificationsSidebarItem />
        <SidebarDivider />
        <SidebarGroup
          label="Settings"
          icon={<UserSettingsSignInAvatar />}
          to="/settings"
        >
          <SidebarSettings />
        </SidebarGroup>
        <SidebarItem
          icon={
            profile?.picture
              ? () => (
                  <Avatar
                    picture={profile?.picture}
                    customStyles={{ width: 24, height: 24, margin: 0 }}
                  />
                )
              : // Fallback icon if no picture, though Avatar handles this internally too
                () => (
                  <Avatar
                    displayName={
                      profile?.displayName || profile?.email || 'User'
                    }
                    customStyles={{ width: 24, height: 24, margin: 0 }}
                  />
                )
          }
          text={profile?.displayName || profile?.email || 'User'}
          to="/settings"
        />
      </Sidebar>
      {children}
    </SidebarPage>
  );
};
