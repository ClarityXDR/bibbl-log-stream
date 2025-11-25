import React, { useState, useCallback } from 'react';
import {
  Box,
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Typography,
  IconButton,
  Tooltip,
  Divider,
  alpha,
  useTheme,
  Collapse,
  Badge,
  Avatar,
} from '@mui/material';
import {
  Dashboard,
  RocketLaunch,
  Input as InputIcon,
  Transform,
  Output,
  Timeline,
  Cloud,
  Settings,
  ChevronLeft,
  ChevronRight,
  DarkMode,
  LightMode,
  Notifications,
  Help,
  ExpandLess,
  ExpandMore,
  Speed,
  Security,
  Storage,
} from '@mui/icons-material';
import { colors } from '../../theme/theme2026';

interface NavItem {
  id: string;
  label: string;
  icon: React.ReactNode;
  badge?: number | string;
  children?: NavItem[];
}

interface AppShellProps {
  children: React.ReactNode;
  activeTab: string;
  onTabChange: (tab: string) => void;
  onThemeToggle?: () => void;
  isDarkMode?: boolean;
}

const DRAWER_WIDTH = 260;
const COLLAPSED_WIDTH = 72;

const NAV_ITEMS: NavItem[] = [
  { id: 'home', label: 'Dashboard', icon: <Dashboard /> },
  { id: 'setup', label: 'Quick Setup', icon: <RocketLaunch />, badge: 'NEW' },
  { id: 'sources', label: 'Sources', icon: <InputIcon /> },
  { id: 'transform', label: 'Transform', icon: <Transform /> },
  { id: 'destinations', label: 'Destinations', icon: <Output /> },
  { id: 'logevents', label: 'Metrics', icon: <Timeline /> },
  { id: 'azure', label: 'Azure', icon: <Cloud /> },
];

const SECONDARY_NAV: NavItem[] = [
  { id: 'settings', label: 'Settings', icon: <Settings /> },
  { id: 'help', label: 'Help & Docs', icon: <Help /> },
];

export default function AppShell({
  children,
  activeTab,
  onTabChange,
  onThemeToggle,
  isDarkMode = true,
}: AppShellProps) {
  const theme = useTheme();
  const [collapsed, setCollapsed] = useState(false);
  const [expandedItems, setExpandedItems] = useState<string[]>([]);

  const toggleCollapse = useCallback(() => {
    setCollapsed((prev) => !prev);
  }, []);

  const toggleExpand = useCallback((itemId: string) => {
    setExpandedItems((prev) =>
      prev.includes(itemId)
        ? prev.filter((id) => id !== itemId)
        : [...prev, itemId]
    );
  }, []);

  const handleNavClick = (item: NavItem) => {
    if (item.children) {
      toggleExpand(item.id);
    } else {
      onTabChange(item.id);
    }
  };

  const renderNavItem = (item: NavItem, depth = 0) => {
    const isActive = activeTab === item.id;
    const isExpanded = expandedItems.includes(item.id);
    const hasChildren = item.children && item.children.length > 0;

    return (
      <React.Fragment key={item.id}>
        <ListItem disablePadding sx={{ display: 'block', mb: 0.5 }}>
          <Tooltip
            title={collapsed ? item.label : ''}
            placement="right"
            arrow
          >
            <ListItemButton
              onClick={() => handleNavClick(item)}
              sx={{
                minHeight: 44,
                px: collapsed ? 2 : 2.5,
                mx: 1,
                borderRadius: 2,
                transition: 'all 0.2s cubic-bezier(0.4, 0, 0.2, 1)',
                pl: depth > 0 ? 4 + depth : undefined,
                ...(isActive && {
                  background: `linear-gradient(135deg, ${alpha(colors.primary[500], 0.15)} 0%, ${alpha(colors.primary[600], 0.1)} 100%)`,
                  borderLeft: `3px solid ${colors.primary[500]}`,
                  '&:hover': {
                    background: `linear-gradient(135deg, ${alpha(colors.primary[500], 0.2)} 0%, ${alpha(colors.primary[600], 0.15)} 100%)`,
                  },
                }),
                ...(!isActive && {
                  '&:hover': {
                    background: alpha(colors.slate[400], 0.08),
                  },
                }),
              }}
            >
              <ListItemIcon
                sx={{
                  minWidth: 0,
                  mr: collapsed ? 0 : 2,
                  justifyContent: 'center',
                  color: isActive ? colors.primary[400] : colors.slate[400],
                  transition: 'color 0.2s ease',
                }}
              >
                {item.badge ? (
                  <Badge
                    badgeContent={item.badge}
                    color="primary"
                    sx={{
                      '& .MuiBadge-badge': {
                        fontSize: '0.6rem',
                        fontWeight: 700,
                        height: 16,
                        minWidth: 16,
                        padding: '0 4px',
                        background: `linear-gradient(135deg, ${colors.primary[500]} 0%, ${colors.primary[600]} 100%)`,
                      },
                    }}
                  >
                    {item.icon}
                  </Badge>
                ) : (
                  item.icon
                )}
              </ListItemIcon>
              {!collapsed && (
                <>
                  <ListItemText
                    primary={item.label}
                    primaryTypographyProps={{
                      fontSize: '0.875rem',
                      fontWeight: isActive ? 600 : 500,
                      color: isActive ? colors.primary[300] : 'text.primary',
                    }}
                  />
                  {hasChildren && (
                    isExpanded ? <ExpandLess /> : <ExpandMore />
                  )}
                </>
              )}
            </ListItemButton>
          </Tooltip>
        </ListItem>
        {hasChildren && !collapsed && (
          <Collapse in={isExpanded} timeout="auto" unmountOnExit>
            <List component="div" disablePadding>
              {item.children!.map((child) => renderNavItem(child, depth + 1))}
            </List>
          </Collapse>
        )}
      </React.Fragment>
    );
  };

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      {/* Sidebar */}
      <Drawer
        variant="permanent"
        sx={{
          width: collapsed ? COLLAPSED_WIDTH : DRAWER_WIDTH,
          flexShrink: 0,
          transition: 'width 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
          '& .MuiDrawer-paper': {
            width: collapsed ? COLLAPSED_WIDTH : DRAWER_WIDTH,
            boxSizing: 'border-box',
            background: `linear-gradient(180deg, ${alpha(colors.slate[900], 0.98)} 0%, ${colors.slate[950]} 100%)`,
            borderRight: `1px solid ${alpha(colors.slate[400], 0.08)}`,
            transition: 'width 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
            overflowX: 'hidden',
          },
        }}
      >
        {/* Logo Section */}
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1.5,
            p: 2,
            minHeight: 68,
            borderBottom: `1px solid ${alpha(colors.slate[400], 0.08)}`,
          }}
        >
          <Box
            sx={{
              width: 36,
              height: 36,
              borderRadius: 2,
              background: `linear-gradient(135deg, ${colors.primary[500]} 0%, ${colors.primary[600]} 100%)`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: `0 4px 12px ${alpha(colors.primary[500], 0.3)}`,
            }}
          >
            <img
              src="/logo.svg"
              alt="Bibbl"
              style={{ width: 24, height: 24, filter: 'brightness(0) invert(1)' }}
            />
          </Box>
          {!collapsed && (
            <Box sx={{ flex: 1 }}>
              <Typography
                variant="h6"
                sx={{
                  fontWeight: 700,
                  background: `linear-gradient(135deg, ${colors.primary[300]} 0%, ${colors.primary[400]} 100%)`,
                  backgroundClip: 'text',
                  WebkitBackgroundClip: 'text',
                  color: 'transparent',
                  letterSpacing: '-0.02em',
                }}
              >
                Bibbl
              </Typography>
              <Typography
                variant="caption"
                sx={{
                  color: colors.slate[500],
                  display: 'block',
                  fontSize: '0.65rem',
                  letterSpacing: '0.1em',
                  textTransform: 'uppercase',
                }}
              >
                Log Stream
              </Typography>
            </Box>
          )}
        </Box>

        {/* Main Navigation */}
        <Box sx={{ flex: 1, py: 2, overflowY: 'auto', overflowX: 'hidden' }}>
          <List sx={{ px: 0.5 }}>
            {NAV_ITEMS.map((item) => renderNavItem(item))}
          </List>
        </Box>

        <Divider sx={{ mx: 2, borderColor: alpha(colors.slate[400], 0.08) }} />

        {/* Secondary Navigation */}
        <List sx={{ px: 0.5, py: 1 }}>
          {SECONDARY_NAV.map((item) => renderNavItem(item))}
        </List>

        {/* Footer Actions */}
        <Box
          sx={{
            p: 2,
            borderTop: `1px solid ${alpha(colors.slate[400], 0.08)}`,
            display: 'flex',
            alignItems: 'center',
            gap: 1,
            justifyContent: collapsed ? 'center' : 'flex-start',
          }}
        >
          <Tooltip title={isDarkMode ? 'Light Mode' : 'Dark Mode'} placement="right">
            <IconButton
              onClick={onThemeToggle}
              size="small"
              sx={{
                color: colors.slate[400],
                '&:hover': {
                  color: colors.primary[400],
                  bgcolor: alpha(colors.primary[500], 0.1),
                },
              }}
            >
              {isDarkMode ? <LightMode fontSize="small" /> : <DarkMode fontSize="small" />}
            </IconButton>
          </Tooltip>

          {!collapsed && (
            <>
              <Tooltip title="Notifications" placement="right">
                <IconButton
                  size="small"
                  sx={{
                    color: colors.slate[400],
                    '&:hover': {
                      color: colors.primary[400],
                      bgcolor: alpha(colors.primary[500], 0.1),
                    },
                  }}
                >
                  <Badge
                    badgeContent={3}
                    color="error"
                    sx={{
                      '& .MuiBadge-badge': {
                        fontSize: '0.6rem',
                        height: 14,
                        minWidth: 14,
                      },
                    }}
                  >
                    <Notifications fontSize="small" />
                  </Badge>
                </IconButton>
              </Tooltip>
              <Box sx={{ flex: 1 }} />
            </>
          )}

          <Tooltip title={collapsed ? 'Expand' : 'Collapse'} placement="right">
            <IconButton
              onClick={toggleCollapse}
              size="small"
              sx={{
                color: colors.slate[400],
                '&:hover': {
                  color: colors.primary[400],
                  bgcolor: alpha(colors.primary[500], 0.1),
                },
              }}
            >
              {collapsed ? (
                <ChevronRight fontSize="small" />
              ) : (
                <ChevronLeft fontSize="small" />
              )}
            </IconButton>
          </Tooltip>
        </Box>
      </Drawer>

      {/* Main Content */}
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          minHeight: '100vh',
          background: `radial-gradient(ellipse 80% 50% at 50% -20%, ${alpha(colors.primary[500], 0.08)} 0%, transparent 50%), ${colors.slate[950]}`,
          overflow: 'auto',
        }}
      >
        {/* Top Bar */}
        <Box
          sx={{
            position: 'sticky',
            top: 0,
            zIndex: 10,
            px: 3,
            py: 2,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            backdropFilter: 'blur(12px)',
            background: alpha(colors.slate[950], 0.8),
            borderBottom: `1px solid ${alpha(colors.slate[400], 0.06)}`,
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            {/* Breadcrumb or page title can go here */}
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            {/* Status indicators */}
            <Box
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1,
                px: 2,
                py: 0.75,
                borderRadius: 2,
                background: alpha(colors.success[500], 0.1),
                border: `1px solid ${alpha(colors.success[500], 0.2)}`,
              }}
            >
              <Box
                sx={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  bgcolor: colors.success[400],
                  boxShadow: `0 0 8px ${colors.success[400]}`,
                  animation: 'pulse 2s infinite',
                  '@keyframes pulse': {
                    '0%, 100%': { opacity: 1 },
                    '50%': { opacity: 0.5 },
                  },
                }}
              />
              <Typography
                variant="caption"
                sx={{
                  color: colors.success[400],
                  fontWeight: 600,
                  letterSpacing: '0.02em',
                }}
              >
                System Healthy
              </Typography>
            </Box>

            <Tooltip title="Quick Actions" placement="bottom">
              <IconButton
                sx={{
                  color: colors.slate[400],
                  '&:hover': { color: colors.primary[400] },
                }}
              >
                <Speed fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        {/* Page Content */}
        <Box sx={{ p: 3 }}>{children}</Box>
      </Box>
    </Box>
  );
}
