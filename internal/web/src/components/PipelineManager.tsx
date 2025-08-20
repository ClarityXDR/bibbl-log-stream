import React, { useState, useEffect } from 'react';
import { Tabs, Tab, Box, Paper, Typography } from '@mui/material';
import SourcesConfig from './SourcesConfig';
import RoutesConfig from './RoutesConfig';
import PipelinesConfig from './PipelinesConfig';
import DestinationsConfig from './DestinationsConfig';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`pipeline-tabpanel-${index}`}
      aria-labelledby={`pipeline-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

export default function PipelineManager() {
  const [value, setValue] = useState(0);

  const handleChange = (event: React.SyntheticEvent, newValue: number) => {
    setValue(newValue);
  };

  return (
    <Paper sx={{ width: '100%' }}>
      <Typography variant="h4" sx={{ p: 2 }}>
        Pipeline Configuration
      </Typography>
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={value} onChange={handleChange} aria-label="pipeline tabs">
          <Tab label="Sources" />
          <Tab label="Routes" />
          <Tab label="Pipelines" />
          <Tab label="Destinations" />
        </Tabs>
      </Box>
      <TabPanel value={value} index={0}>
        <SourcesConfig />
      </TabPanel>
      <TabPanel value={value} index={1}>
        <RoutesConfig />
      </TabPanel>
      <TabPanel value={value} index={2}>
        <PipelinesConfig />
      </TabPanel>
      <TabPanel value={value} index={3}>
        <DestinationsConfig />
      </TabPanel>
    </Paper>
  );
}
