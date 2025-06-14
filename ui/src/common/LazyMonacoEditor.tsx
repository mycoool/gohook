import React, { Suspense, lazy } from 'react';
import { CircularProgress, Box, Typography } from '@material-ui/core';

// 懒加载Monaco Editor
const MonacoEditor = lazy(() => import('react-monaco-editor'));

interface LazyMonacoEditorProps {
  width?: string | number;
  height?: string | number;
  language?: string;
  theme?: string;
  value: string;
  options?: Record<string, unknown>;
  onChange?: (value: string) => void;
}

const LazyMonacoEditor: React.FC<LazyMonacoEditorProps> = (props) => (
  <Suspense 
    fallback={
      <Box 
        display="flex" 
        justifyContent="center" 
        alignItems="center" 
        flexDirection="column"
        height={props.height || '100%'}
        minHeight={200}
      >
        <CircularProgress size={32} />
        <Typography variant="body2" style={{ marginTop: 16 }}>
          正在加载编辑器...
        </Typography>
      </Box>
    }
  >
    <MonacoEditor {...props} />
  </Suspense>
);

export default LazyMonacoEditor; 