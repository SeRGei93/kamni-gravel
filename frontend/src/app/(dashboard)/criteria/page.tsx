'use client';

import { useEffect, useState } from 'react';
import { criteriaApi } from '@/api/criteria';
import type {
  Criteria,
  CreateCriteriaRequest,
  UpdateCriteriaRequest,
} from '@/types';
import CriteriaTable from '@/components/criteria/CriteriaTable';
import CriteriaForm from '@/components/criteria/CriteriaForm';
import Button from '@/components/ui/button/Button';
import { Modal } from '@/components/ui/modal';
import { useModal } from '@/hooks/useModal';
import { PlusIcon } from '@/icons';

export default function CriteriaPage() {
  const [criteria, setCriteria] = useState<Criteria[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [editingCriteria, setEditingCriteria] = useState<Criteria | null>(null);

  const { isOpen: isCreateOpen, openModal: openCreateModal, closeModal: closeCreateModal } =
    useModal();
  const { isOpen: isEditOpen, openModal: openEditModal, closeModal: closeEditModal } = useModal();

  // Загрузка критериев
  useEffect(() => {
    loadCriteria();
  }, []);

  const loadCriteria = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await criteriaApi.getAll();
      setCriteria(response.criteria);
    } catch (err) {
      setError('Ошибка загрузки критериев');
      console.error('Failed to load criteria:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreate = async (data: CreateCriteriaRequest | UpdateCriteriaRequest) => {
    try {
      if (editingCriteria) {
        await criteriaApi.update(editingCriteria.id, data as UpdateCriteriaRequest);
      } else {
        await criteriaApi.create(data as CreateCriteriaRequest);
      }
      closeCreateModal();
      closeEditModal();
      setEditingCriteria(null);
      await loadCriteria();
    } catch (err) {
      setError(editingCriteria ? 'Ошибка обновления критерия' : 'Ошибка создания критерия');
      console.error('Failed to save criteria:', err);
      throw err;
    }
  };

  const handleEdit = (criterion: Criteria) => {
    setEditingCriteria(criterion);
    openEditModal();
  };

  const handleDelete = async (criteriaId: number) => {
    try {
      await criteriaApi.delete(criteriaId);
      await loadCriteria();
    } catch (err) {
      setError('Ошибка удаления критерия');
      console.error('Failed to delete criteria:', err);
    }
  };

  const handleCancelCreate = () => {
    closeCreateModal();
  };

  const handleCancelEdit = () => {
    closeEditModal();
    setEditingCriteria(null);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
            Критерии
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Управление критериями для призов
          </p>
        </div>
        <Button
          startIcon={<PlusIcon />}
          onClick={openCreateModal}
          size="sm"
        >
          Создать критерий
        </Button>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {/* Информация о количестве */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Найдено критериев: {criteria.length}
        </p>
      </div>

      {/* Таблица критериев */}
      <CriteriaTable
        criteria={criteria}
        isLoading={isLoading}
        onEdit={handleEdit}
        onDelete={handleDelete}
      />

      {/* Модальное окно создания */}
      <Modal isOpen={isCreateOpen} onClose={closeCreateModal} className="max-w-2xl m-4">
        <div className="no-scrollbar relative w-full max-w-2xl overflow-y-auto rounded-3xl bg-white p-4 dark:bg-gray-900 lg:p-11">
          <div className="px-2 pr-14">
            <h4 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white/90">
              Создать критерий
            </h4>
            <p className="mb-6 text-sm text-gray-500 dark:text-gray-400 lg:mb-7">
              Заполните форму для создания нового критерия
            </p>
          </div>
          <div className="px-2">
            <CriteriaForm
              onSubmit={handleCreate}
              onCancel={handleCancelCreate}
            />
          </div>
        </div>
      </Modal>

      {/* Модальное окно редактирования */}
      <Modal isOpen={isEditOpen} onClose={closeEditModal} className="max-w-2xl m-4">
        <div className="no-scrollbar relative w-full max-w-2xl overflow-y-auto rounded-3xl bg-white p-4 dark:bg-gray-900 lg:p-11">
          <div className="px-2 pr-14">
            <h4 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white/90">
              Редактировать критерий
            </h4>
            <p className="mb-6 text-sm text-gray-500 dark:text-gray-400 lg:mb-7">
              Измените данные критерия
            </p>
          </div>
          <div className="px-2">
            {editingCriteria && (
              <CriteriaForm
                criteria={editingCriteria}
                onSubmit={handleCreate}
                onCancel={handleCancelEdit}
              />
            )}
          </div>
        </div>
      </Modal>
    </div>
  );
}
