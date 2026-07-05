import { SecretDto } from '../../api';
import Table from '@mui/material/Table';
import TableContainer from '@mui/material/TableContainer';
import Paper from '@mui/material/Paper';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Link from '@mui/material/Link';
import LinearProgress from '@mui/material/LinearProgress';
import IconButton from '@mui/material/IconButton';
import DeleteIcon from '@mui/icons-material/Delete';
import React from 'react';
import { useNavigate } from 'react-router-dom';

type SecretTableProps = {
  data: SecretDto[];
  loading: boolean;
  page: number;
  rowsPerPage: number;
  hasNextPage: boolean;
  onPageChange: (
    event: React.MouseEvent<HTMLButtonElement> | null,
    newPage: number,
  ) => void;
  onRowsPerPageChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  onDeleteSecret: (secretName: string) => void;
};

export const SecretTable = ({
  data,
  loading,
  onDeleteSecret,
}: SecretTableProps) => {
  const navigate = useNavigate();

  return (
    <Paper>
      <TableContainer>
        {loading && <LinearProgress />}
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Secret Name</TableCell>
              <TableCell align="right">Namespace</TableCell>
              <TableCell align="right">Created Date</TableCell>
              <TableCell align="center" style={{ width: 100 }}>
                Actions
              </TableCell>
            </TableRow>
          </TableHead>

          <TableBody>
            {data.length === 0 && !loading && (
              <TableRow>
                <TableCell colSpan={4} align="center">
                  No secrets found
                </TableCell>
              </TableRow>
            )}
            {data.map(row => (
              <TableRow key={row.name} hover>
                <TableCell component="th" scope="row">
                  <Link
                    component="button"
                    onClick={() => navigate(`${row.name}`)}
                  >
                    {row.name}
                  </Link>
                </TableCell>
                <TableCell align="right">{row.namespace}</TableCell>
                <TableCell align="right">
                  {row.createdAt
                    ? new Date(row.createdAt).toLocaleString()
                    : 'N/A'}
                </TableCell>
                <TableCell align="center">
                  <IconButton
                    aria-label="delete"
                    color="error"
                    onClick={() => onDeleteSecret(row.name)}
                    disabled={loading}
                    size="small"
                  >
                    <DeleteIcon />
                  </IconButton>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      {/*<TablePagination*/}
      {/*  component="div"*/}
      {/*  count={totalCount}*/}
      {/*  page={page}*/}
      {/*  onPageChange={onPageChange}*/}
      {/*  rowsPerPage={rowsPerPage}*/}
      {/*  onRowsPerPageChange={onRowsPerPageChange}*/}
      {/*  rowsPerPageOptions={[5, 10, 20]}*/}
      {/*/>*/}
    </Paper>
  );
};
