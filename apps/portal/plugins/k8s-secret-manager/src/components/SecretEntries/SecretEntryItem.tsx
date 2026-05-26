import { Grid, IconButton, InputAdornment, OutlinedInput } from '@mui/material';
import React, { useState } from 'react';
import { Visibility, VisibilityOff } from '@material-ui/icons';
import DeleteIcon from '@mui/icons-material/Delete';
import CloseIcon from '@mui/icons-material/Close';
import EditIcon from '@mui/icons-material/Edit';
import CheckIcon from '@mui/icons-material/Check';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogActions from '@mui/material/DialogActions';

interface SecretEntryItemProps {
  entryKey: string;
  entryValue: string;
  onUpdate: (key: string, value: string) => Promise<void>;
  onDelete: (key: string) => Promise<void>;
}

export const SecretEntryItem = ({
  entryKey,
  entryValue,
  onUpdate,
  onDelete,
}: SecretEntryItemProps) => {
  const [showEnvValue, setShowEnvValue] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(entryValue);
  const [isSaving, setIsSaving] = useState(false);
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);

  const handleSave = async () => {
    setIsSaving(true);
    await onUpdate(entryKey, editValue);
    setIsSaving(false);
    setIsEditing(false);
    setShowEnvValue(false);
  };

  const handleCancel = () => {
    setEditValue(entryValue);
    setIsEditing(false);
  };

  const handleConfirmDelete = async () => {
    setOpenDeleteDialog(false);
    await onDelete(entryKey);
  };

  return (
    <>
      <Grid container sx={{ marginBottom: '12px' }}>
        <Grid item xs={5}>
          <OutlinedInput
            size="small"
            disabled
            value={entryKey}
            sx={{ width: '100%' }}
          />
        </Grid>

        <Grid item xs={6}>
          <OutlinedInput
            size="small"
            type={
              showEnvValue && !isEditing
                ? 'text'
                : isEditing
                ? 'text'
                : 'password'
            }
            disabled={!isEditing || isSaving}
            value={isEditing ? editValue : entryValue}
            onChange={e => setEditValue(e.target.value)}
            sx={{ width: '100%' }}
            endAdornment={
              !isEditing && (
                <InputAdornment position="end">
                  <IconButton
                    onClick={() => setShowEnvValue(!showEnvValue)}
                    edge="end"
                  >
                    {showEnvValue ? <VisibilityOff /> : <Visibility />}
                  </IconButton>
                </InputAdornment>
              )
            }
          />
        </Grid>

        <Grid item xs={1} sx={{ display: 'flex', alignItems: 'center' }}>
          {isEditing ? (
            <React.Fragment>
              <IconButton
                color="success"
                onClick={handleSave}
                disabled={isSaving}
              >
                <CheckIcon />
              </IconButton>
              <IconButton
                color="default"
                onClick={handleCancel}
                disabled={isSaving}
              >
                <CloseIcon />
              </IconButton>
            </React.Fragment>
          ) : (
            <React.Fragment>
              <IconButton color="primary" onClick={() => setIsEditing(true)}>
                <EditIcon />
              </IconButton>
              <IconButton
                color="error"
                onClick={() => setOpenDeleteDialog(true)}
              >
                <DeleteIcon />
              </IconButton>
            </React.Fragment>
          )}
        </Grid>
      </Grid>

      {/* Confirmation Dialog */}
      <Dialog
        open={openDeleteDialog}
        onClose={() => setOpenDeleteDialog(false)}
        aria-labelledby="delete-dialog-title"
        aria-describedby="delete-dialog-description"
      >
        <DialogTitle id="delete-dialog-title">Confirm Deletion</DialogTitle>
        <DialogContent>
          <DialogContentText id="delete-dialog-description">
            Are you sure you want to delete the secret{' '}
            <strong>{entryKey}</strong>? This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDeleteDialog(false)} color="inherit">
            Cancel
          </Button>
          <Button
            onClick={handleConfirmDelete}
            color="error"
            variant="contained"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};
