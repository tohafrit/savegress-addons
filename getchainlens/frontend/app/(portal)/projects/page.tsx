'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import {
  Folder,
  Plus,
  Search,
  MoreVertical,
  Settings,
  Trash2,
  FileCode2,
  Shield,
  AlertTriangle,
} from 'lucide-react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Button,
  Input,
  Badge,
} from '@/components/ui';
import { api } from '@/lib/api';
import { formatDate, formatTimeAgo } from '@/lib/utils';
import type { Project } from '@/types';

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);

  useEffect(() => {
    async function loadProjects() {
      const result = await api.getProjects();
      if (result.data) {
        setProjects(result.data);
      }
      setIsLoading(false);
    }
    loadProjects();
  }, []);

  const filteredProjects = projects.filter((project) =>
    project.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleDelete = async (projectId: string) => {
    if (!confirm('Are you sure you want to delete this project? All contracts will be unassigned.')) return;

    const result = await api.deleteProject(projectId);
    if (!result.error) {
      setProjects(projects.filter((p) => p.id !== projectId));
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-accent-cyan border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white">Projects</h1>
          <p className="mt-1 text-gray-400">
            Organize your contracts into projects
          </p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Project
        </Button>
      </div>

      {/* Search */}
      <Card>
        <CardContent className="p-4">
          <Input
            placeholder="Search projects..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            icon={<Search className="h-4 w-4" />}
          />
        </CardContent>
      </Card>

      {/* Projects grid */}
      {filteredProjects.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filteredProjects.map((project) => (
            <ProjectCard
              key={project.id}
              project={project}
              onDelete={() => handleDelete(project.id)}
            />
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <Folder className="mx-auto h-12 w-12 text-gray-600" />
            <h3 className="mt-4 text-lg font-medium text-white">No projects found</h3>
            <p className="mt-2 text-gray-400">
              {searchQuery
                ? 'Try adjusting your search'
                : 'Create your first project to organize your contracts'}
            </p>
            {!searchQuery && (
              <Button className="mt-4" onClick={() => setShowCreateModal(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Create Project
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Create Project Modal */}
      {showCreateModal && (
        <CreateProjectModal
          onClose={() => setShowCreateModal(false)}
          onSuccess={(project) => {
            setProjects([project, ...projects]);
            setShowCreateModal(false);
          }}
        />
      )}
    </div>
  );
}

function ProjectCard({
  project,
  onDelete,
}: {
  project: Project;
  onDelete: () => void;
}) {
  // Mock data for demo
  const contractCount = Math.floor(Math.random() * 10) + 1;
  const issueCount = Math.floor(Math.random() * 25);
  const avgScore = Math.floor(Math.random() * 30) + 70;

  return (
    <Card hover className="relative overflow-hidden">
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/20">
              <Folder className="h-6 w-6 text-accent-cyan" />
            </div>
            <div>
              <h3 className="font-semibold text-white">{project.name}</h3>
              <p className="text-sm text-gray-400">
                Created {formatTimeAgo(new Date(project.created_at))}
              </p>
            </div>
          </div>
          <button
            onClick={onDelete}
            className="p-1.5 rounded-lg text-gray-500 hover:text-severity-critical hover:bg-severity-critical/10 transition-colors"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>

        {project.description && (
          <p className="mt-3 text-sm text-gray-400 line-clamp-2">
            {project.description}
          </p>
        )}

        <div className="mt-4 grid grid-cols-3 gap-4 pt-4 border-t border-gray-800">
          <div className="text-center">
            <div className="flex items-center justify-center gap-1">
              <FileCode2 className="h-4 w-4 text-gray-500" />
              <span className="text-lg font-semibold text-white">{contractCount}</span>
            </div>
            <p className="text-xs text-gray-500">Contracts</p>
          </div>
          <div className="text-center">
            <div className="flex items-center justify-center gap-1">
              <AlertTriangle className="h-4 w-4 text-gray-500" />
              <span className="text-lg font-semibold text-white">{issueCount}</span>
            </div>
            <p className="text-xs text-gray-500">Issues</p>
          </div>
          <div className="text-center">
            <div className="flex items-center justify-center gap-1">
              <Shield className="h-4 w-4 text-gray-500" />
              <span className="text-lg font-semibold text-white">{avgScore}</span>
            </div>
            <p className="text-xs text-gray-500">Avg Score</p>
          </div>
        </div>

        <div className="mt-4 flex gap-2">
          <Button variant="secondary" size="sm" className="flex-1" asChild>
            <Link href={`/projects/${project.id}`}>
              View Project
            </Link>
          </Button>
          <Button variant="ghost" size="icon-sm" asChild>
            <Link href={`/projects/${project.id}/settings`}>
              <Settings className="h-4 w-4" />
            </Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function CreateProjectModal({
  onClose,
  onSuccess,
}: {
  onClose: () => void;
  onSuccess: (project: Project) => void;
}) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    const result = await api.createProject(name, description);

    if (result.data) {
      onSuccess(result.data);
    } else {
      setError(result.error || 'Failed to create project');
    }

    setIsLoading(false);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Create Project</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="rounded-lg bg-severity-critical/10 border border-severity-critical/30 p-3 text-sm text-severity-critical">
                {error}
              </div>
            )}

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Name</label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="My Project"
                required
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Description (optional)</label>
              <Input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="A brief description of your project"
              />
            </div>

            <div className="flex gap-3 pt-4">
              <Button type="button" variant="secondary" onClick={onClose} className="flex-1">
                Cancel
              </Button>
              <Button type="submit" isLoading={isLoading} className="flex-1">
                Create Project
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
